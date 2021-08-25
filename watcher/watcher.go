// Copyright 2021 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package watcher

import (
	"context"
	"sync"

	"github.com/cenkalti/backoff/v4"
	"github.com/thediveo/whalewatcher"
	"github.com/thediveo/whalewatcher/engineclient"
)

// Watcher allows keeping track of the currently alive containers of a container
// engine, optionally with the composer projects they're associated with (if
// supported).
type Watcher interface {
	// Portfolio returns the current portfolio for reading. During
	// resynchronization with a container engine this can be the buffered
	// portfolio until the watcher has caught up with the new state after an
	// engine reconnect. For this reason callers must not keep the returned
	// Portfolio reference for longer periods of time, but just for what they
	// immediately need to query a Portfolio for.
	Portfolio() *whalewatcher.Portfolio
	// Ready returns a channel that gets closed after the initial
	// synchronization has been achieved. Watcher clients do not need to wait
	// for the Ready channel to get closed to work with the portfolio; this just
	// helps those applications that need to wait for results as opposed to take
	// whatever information currently is available, or not.
	Ready() <-chan struct{}
	// Watch synchronizes the Portfolio to the connected container engine's
	// state with respect to alive containers and then continuously watches for
	// changes. Watch only returns after the specified context has been
	// cancelled. It will automatically reconnect in case of loss of connection
	// to the connected container engine.
	Watch(ctx context.Context) error
	// ID returns the (more or less) unique engine identifier; the exact format
	// is engine-specific.
	ID(ctx context.Context) string
	// Identifier of the type of container engine, such as "docker.com",
	// "containerd.io", et cetera.
	Type() string
	// Container engine API path.
	API() string
	// Container engine PID, when known.
	PID() int
	// Underlying engine client (engine-specific).
	Client() interface{}
	// Close cleans up and release any engine client resources, if necessary.
	Close()
}

// watcher watches a Docker daemon for containers to become alive and later
// die, keeping track as well as automatically synchronizing at start and after
// reconnects. Please note that not the whole container lifecycle gets monitored
// but only the phase(s) where a container has either running or frozen
// container processes.
type watcher struct {
	engine    engineclient.EngineClient // container engine (adaptor)
	buggeroff backoff.BackOff

	pfmux          sync.RWMutex            // supports make-before-break during resync.
	readportfolio  *whalewatcher.Portfolio // portfolio as seen by object users.
	writeportfolio *whalewatcher.Portfolio // portfolio we're updating.

	eventgate      sync.Mutex         // not a RWMutex as it doesn't buy us anything here.
	listinprogress bool               // listing containers in progress.
	bluenorwegians []string           // container IDs we know to have died while list in progress.
	pauses         pendingPauseStates // (un)pause state changes while list in progress.

	once  sync.Once     // "protects" the ready channel
	ready chan struct{} // ready channel signal
}

// New returns a new Watcher tracking alive containers as they come and go,
// using the specified container EngineClient. If the backoff is nil then the
// backoff defaults to backoff.StopBackOff, that is, any failed operation will
// never be retried.
func New(engine engineclient.EngineClient, buggeroff backoff.BackOff) Watcher {
	pf := whalewatcher.NewPortfolio()
	if buggeroff == nil {
		buggeroff = &backoff.StopBackOff{}
	}
	return &watcher{
		engine:         engine,
		buggeroff:      buggeroff,
		readportfolio:  pf,
		writeportfolio: pf,
		ready:          make(chan struct{}),
	}
}

// Portfolio returns the current portfolio for reading. During resynchronization
// with a container engine this can be the buffered portfolio until the watcher
// has caught up with the new state after an engine reconnect. For this reason
// callers must not keep the returned Portfolio reference for longer periods of
// time, but just for what they immediately need to query a Portfolio for.
func (ww *watcher) Portfolio() *whalewatcher.Portfolio {
	ww.pfmux.RLock()
	defer ww.pfmux.RUnlock()
	return ww.readportfolio
}

// Ready returns a channel that gets closed after the initial
// synchronization has been achieved. Watcher clients do not need to wait
// for the Ready channel to get closed to work with the portfolio; this just
// helps those applications that need to wait for results as opposed to take
// whatever information currently is available, or not.
func (ww *watcher) Ready() <-chan struct{} {
	return ww.ready
}

// ID returns the (more or less) unique engine identifier; the exact format is
// engine-specific.
func (ww *watcher) ID(ctx context.Context) string {
	return ww.engine.ID(ctx)
}

// Identifier of the type of container engine, such as "docker.com",
// "containerd.io", et cetera.
func (ww *watcher) Type() string { return ww.engine.Type() }

// Container engine API path.
func (ww *watcher) API() string { return ww.engine.API() }

// Container engine PID, when known.
func (ww *watcher) PID() int { return ww.engine.PID() }

// Client returns the underlying engine client (engine-specific).
func (ww *watcher) Client() interface{} { return ww.engine.Client() }

// Close cleans up and release any underlying engine client resources, if
// necessary.
func (ww *watcher) Close() {
	ww.engine.Close()
}

// Watch synchronizes the Portfolio to the connected container engine's state
// with respect to alive containers and then continuously watches for changes.
// Watch only returns after the specified context has been cancelled. It will
// automatically reconnect in case of loss of connection to the connected
// container engine, subject to the backoff (and thus optional throttling or
// rate-limiting) specified when this watch was created.
func (ww *watcher) Watch(ctx context.Context) error {
	return backoff.Retry(func() error {
		// In case we have an existing and non-empty portfolio, keep that
		// visible to our users while we try to synchronize. If not, then simply
		// go "live" immediately.
		ww.pfmux.Lock()
		if ww.writeportfolio.ContainerTotal() != 0 {
			ww.writeportfolio = whalewatcher.NewPortfolio()
		}
		if ww.readportfolio.ContainerTotal() == 0 {
			ww.readportfolio = ww.writeportfolio
		}
		ww.pfmux.Unlock()
		// Start receiving container-related events and also fire off a list of
		// containers query. Subscribing to events always succeeds but may then
		// result in the error channel (immediately) becoming readable which
		// we'll catch only later below.
		//
		// We also create a child context that can be can be cancelled without
		// cancelling the parent context: this is needed in case the list
		// operation utterly fails and we thus need to cancel listing for
		// container events, too. Unfortunately, govet totally go bonkers with
		// their less-than-stellar "heuristics", thinking that we "leak" a
		// cancel ... which we don't. When the parent got cancelled, we simply
		// cannot "leak" a child cancel, whatever govet's "opinion" is.
		eventsctx, cancelevents := context.WithCancel(ctx)
		evs, errs := ww.engine.LifecycleEvents(eventsctx)
		listerr := make(chan error)
		go func() {
			if err := ww.list(ctx); err != nil {
				// list failed for some severe reason, so bail out and tell the
				// event listener to abort, too.
				listerr <- err
				return
			}
			// Bring the synchronized portfolio "online" so that object users
			// can now see the current portfolio and not the "still" portfolio.
			ww.pfmux.Lock()
			ww.readportfolio = ww.writeportfolio
			ww.pfmux.Unlock()
		}()
		// Permanently receive and process container lifecycle-related
		// events...
		for {
			select {
			case err := <-errs:
				_ = cancelevents // stupid, stupid govet: lots of stupid opinion, nuts of brainz.
				// The reason of a cancelled context has been flattened into the
				// client's event stream error, grrr. We thus first check on a
				// cancelled (parent) context in case of any event stream error
				// and let that take priority.
				if ctxerr := ctx.Err(); ctxerr == context.Canceled {
					err = backoff.Permanent(ctxerr)
				}
				return err

			case err := <-listerr:
				// the concurrent list operation has failed so we need to cancel
				// our event binge watching, too. This isn't a permanent error,
				// at least not from the cancelled context perspective.
				cancelevents()
				return err

			case ev := <-evs:
				// Churn events.
				switch ev.Type {
				case engineclient.ContainerStarted:
					ww.born(ctx, ev.ID)
				case engineclient.ContainerExited:
					ww.demised(ev.ID, ev.Project)
				case engineclient.ContainerPaused:
					ww.paused(ev.ID, ev.Project, true)
				case engineclient.ContainerUnpaused:
					ww.paused(ev.ID, ev.Project, false)
				}
			}
		}
	}, ww.buggeroff)
}

// born adds a single container (identified by its unique ID) to our set of
// known live and kicking containers. As we want to store some container details
// (such as container's initial process PID, project, ...).
//
// Note bene: this is just a thin wrapper to mainly ease unit testing.
func (ww *watcher) born(ctx context.Context, id string) {
	cntr, err := ww.engine.Inspect(ctx, id)
	if err == nil {
		// The portfolio already properly handles concurrency operations, so we
		// don't need to take any special care here. However, as we're
		// potentially juggling portfolios around while resynchronizing after
		// loss of the event stream, we must lock access to the correct
		// portfolio for a short period of time.
		ww.pfmux.RLock()
		pf := ww.writeportfolio
		ww.pfmux.RUnlock()
		pf.Add(cntr)
	}
}

// demised removes the "permanently sleeping" container with the specified ID
// from our container portfolio, ensuring it won't pop up again due to an
// overlapping list scan. In case the project name isn't known (such as with the
// containerd engine), the reserved "name" engineclient.ProjectUnknown can be
// passed in and it will be derived automatically.
func (ww *watcher) demised(id string, projectname string) {
	// The "event gate" does not only serializes access to the shared state
	// between the container lifecycle event handler and the container listing
	// one-shot go routine, it also serializes container termination lifecycle
	// events against adding the results of a container list scan, so that
	// adding list results becomes "atomic".
	//
	// Notice that we don't defer the unlock as we don't want to carry out the
	// final removal of the container from our portfolio while still under our
	// gating mutex. We want to keep the lock as short as possible.
	ww.eventgate.Lock()
	// While a full container listing is in progress, we need to remember all
	// the dead containers seen during this phase in order to not accidentally
	// adding them back in case listing and deceasing overlap in unfortunate
	// ways.
	if ww.listinprogress {
		ww.bluenorwegians = append(ww.bluenorwegians, id)
	}
	ww.pauses.Remove(id) // ensure to remove any pending (un)pause state update.
	ww.eventgate.Unlock()
	ww.pfmux.RLock()
	pf := ww.writeportfolio
	ww.pfmux.RUnlock()
	// In case the project is unknown, we need to find the container the hard
	// way. Please note that container objects are considered to be immutable,
	// so we need to update its project accordingly.
	if projectname == engineclient.ProjectUnknown {
		container := pf.Container(id)
		if container == nil {
			return
		}
		projectname = container.Project
	}
	pf.Remove(id, projectname)
}

// paused either updates a container's paused state or schedules for a later
// state update in case a container listing is in progress. In case the project
// name isn't known (such as with the containerd engine), the reserved "name"
// engineclient.ProjectUnknown can be passed in and it will be derived
// automatically.
func (ww *watcher) paused(id string, projectname string, paused bool) {
	ww.eventgate.Lock()
	if ww.listinprogress {
		// while the (initial) listing of containers, including inspecting them,
		// is in progress, we need to pile up any state changes we see: that's
		// because we cannot know when exactly the state of a container was
		// listed (or rather, inspected) and how it relates to the pausing state
		// changes. To make things worse, containerd users do not necessarily
		// use the container IDs as UIDs, with nerdctl being a bad example,
		// while Docker does the right thing using UIDs (but dropping the name
		// information at the containerd level, sigh).
		ww.pauses.Add(id, paused)
		ww.eventgate.Unlock()
		return
	}
	ww.eventgate.Unlock()
	// We can update the pause status of a container directly.
	ww.pfmux.RLock()
	pf := ww.writeportfolio
	ww.pfmux.RUnlock()
	// In case the project is unknown, we need to find the container the hard
	// way. Please note that container objects are considered to be immutable,
	// so we need to update its project accordingly.
	if projectname == engineclient.ProjectUnknown {
		container := pf.Container(id)
		if container == nil {
			return
		}
		projectname = container.Project
	}
	if proj := pf.Project(projectname); proj != nil {
		proj.SetPaused(id, paused)
	}
}

// list scans for currently alive and kicking containers and then adds the
// containers found to our container portfolio.
func (ww *watcher) list(ctx context.Context) error {
	// In case any container(s) die while our container list is in progress,
	// make sure to pile up their IDs so we later can skip any already dead
	// containers. This is necessary as while container lifecycle-related events
	// (especially "start" and "die") are properly ordered, fake "start" events
	// from listing containers aren't synchronized and properly ordered with
	// respect to the event log.
	ww.eventgate.Lock()
	ww.listinprogress = true
	ww.eventgate.Unlock()
	// Now try to scan the currently available containers and take only the
	// alive into further consideration. This is a potentially lengthy
	// operation, as we need to inspect each potential candidate individually
	// due to the way container engine APIs tend to be designed.
	alives, err := ww.engine.List(ctx)
	if err != nil {
		ww.once.Do(func() {
			close(ww.ready)
		})
		return err // list? what list??
	}
	// We now lock out any competing container demise events so we can update
	// the portfolio from the list scan results atomically, but still taking
	// into account all those containers that have gone in the time frame where
	// we scanned for alive containers.
	ww.eventgate.Lock()
	defer func() {
		ww.bluenorwegians = []string{}
		ww.pauses = pendingPauseStates{}
		ww.listinprogress = false // not strictly necessary here, but anywhere within the gated zone.
		ww.eventgate.Unlock()
		ww.once.Do(func() {
			close(ww.ready)
		})
	}()
	ww.pfmux.RLock()
	pf := ww.writeportfolio
	ww.pfmux.RUnlock()
nextpet:
	for _, alive := range alives {
		// Did the container die in between...? Then skip it and get another pet.
		for _, bluenorwegian := range ww.bluenorwegians {
			if bluenorwegian == alive.ID {
				continue nextpet
			}
		}
		// Otherwise, add the container we've found in the list to our
		// portfolio; this is "quick" operation without any trips to the
		// container engine (we already did the "slow" and time-consuming bits
		// before, such as inspecting the details).
		pf.Add(alive)
	}
	// Play back any pending pause state changes that occurred while the listing
	// was in progress; if any such pause state changes refer to deceased IDs,
	// then don't care. containerd's architecture of not enforcing UIDs (as
	// opposed to IDs) mainly causes us all this hassle, so that's the reason
	// why we queue state changes and play them back after the
	// listing/inspection has finished.
	//
	// Note: we're still under the eventgate lock.
	for _, pause := range ww.pauses {
		if container := pf.Container(pause.ID); container != nil {
			if project := pf.Project(container.Project); project != nil {
				project.SetPaused(pause.ID, pause.Paused)
			}
		}
	}
	// Tumble into defer'red clearing the list of dead parrots and carrying on.
	return nil
}
