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

package whalewatcher

import (
	"context"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// MobyAPIClient is a Docker client offering the container and system APIs. For
// production, Docker's client.Client is a compatible implementation, for unit
// testing our very own mockingmoby.MockingMoby.
type MobyAPIClient interface {
	client.ContainerAPIClient
	client.SystemAPIClient
}

// Whalewatcher watches a Docker daemon for containers to become alive and later
// die, keeping track as well as automatically synchronizing at start and after
// reconnects. Please note that not the whole container lifecycle gets monitored
// but only the phase(s) where a container has either running or frozen
// container processes.
type Whalewatcher struct {
	moby MobyAPIClient

	pfmux          sync.RWMutex
	readportfolio  *Portfolio // portfolio as seen by object users
	writeportfolio *Portfolio // portfolio we're updating

	eventgate      sync.Mutex // not a RWMutex as it doesn't buy us anything here.
	ongoinglisting bool       // a container list is in progress.
	bluenorwegians []string   // container IDs we know to have died while list in progress.
}

// NewWhalewatcher returns a new Whalewatcher watching for containers to come
// alive or die using the specified Docker API client.
func NewWhalewatcher(moby MobyAPIClient) *Whalewatcher {
	pf := newPortfolio()
	return &Whalewatcher{
		moby:           moby,
		readportfolio:  pf,
		writeportfolio: pf,
	}
}

func (ww *Whalewatcher) Portfolio() *Portfolio {
	ww.pfmux.RLock()
	defer ww.pfmux.RUnlock()
	return ww.readportfolio
}

func (ww *Whalewatcher) Watch(ctx context.Context) {
	evfilters := filters.NewArgs(
		filters.KeyValuePair{Key: "type", Value: "container"},
		filters.KeyValuePair{Key: "event", Value: "start"},
		filters.KeyValuePair{Key: "event", Value: "die"})
	for {
		// In case we have an existing and non-empty portfolio, keep that
		// visible to our users while we try to synchronize. If not, then simply
		// go "live" immediately.
		ww.pfmux.Lock()
		if ww.writeportfolio.ContainerTotal() != 0 {
			ww.writeportfolio = newPortfolio()
		}
		if ww.readportfolio.ContainerTotal() == 0 {
			ww.readportfolio = ww.writeportfolio
		}
		ww.pfmux.Unlock()
		// Start receiving container-related events and also fire off a list of
		// containers query.
		evs, errs := ww.moby.Events(ctx, types.EventsOptions{Filters: evfilters})
		go func() {
			ww.list(ctx)
			// Bring the synchronized portfolio "online" so that object users
			// can now see the current portfolio and not the "still".
			ww.pfmux.Lock()
			ww.readportfolio = ww.writeportfolio
			ww.pfmux.Unlock()
		}()
		var err error
	listentoevents:
		for {
			select {
			case err = <-errs:
				// The reason of a cancelled context has been flattened into the
				// client's event stream error, grrr. We thus first check on a
				// cancelled context in case of any event stream error and let
				// that take priority.
				if ctx.Err() == context.Canceled {
					err = ctx.Err()
				}
				break listentoevents
			case ev := <-evs:
				switch ev.Action {
				case "start":
					ww.born(ctx, ev.Actor.ID)
				case "die":
					ww.demised(ev.Actor.ID, ev.Actor.Attributes[ComposerProjectLabel])
				}
			}
		}
		// The event flow may have ceased either because (1) the context was
		// cancelled or (2) the Docker daemon has disconnected. In case of (2)
		// we want to retry. In case of (1) that's the signal to us that our
		// work's done.
		if err == context.Canceled {
			return
		}
		// Crude rate limiter
		time.Sleep(time.Second * 1)
	}
}

// born adds a single container (identified by its unique ID) to our set of
// known live and kicking containers. As we want to store some container details
// (such as container's initial process PID, project, ...).
//
// Note bene: this is just a thin wrapper to mainly ease unit testing.
func (ww *Whalewatcher) born(ctx context.Context, id string) {
	cntr, err := newContainer(ctx, ww.moby, id)
	if err == nil {
		// The portfolio already properly handles concurrency operations, so we
		// don't need to take any special care here. However, as we're
		// potentially juggling portfolios around while resynchronizing after
		// loss of the event stream, we must lock access to the correct
		// portfolio for a short period of time.
		ww.pfmux.RLock()
		pf := ww.writeportfolio
		ww.pfmux.RUnlock()
		pf.add(cntr)
	}
}

// demised removes the "permanently sleeping" container with the specified ID
// from our container portfolio, ensuring it won't pop up again due to an
// overlapping list scan.
func (ww *Whalewatcher) demised(id string, projectname string) {
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
	if ww.ongoinglisting {
		ww.bluenorwegians = append(ww.bluenorwegians, id)
	}
	ww.eventgate.Unlock()
	ww.pfmux.RLock()
	pf := ww.writeportfolio
	ww.pfmux.RUnlock()
	pf.remove(id, projectname)
}

// list scans for currently alive and kicking containers and then adds the
// containers found to our container portfolio.
func (ww *Whalewatcher) list(ctx context.Context) {
	// In case any container(s) die while our container list is in progress,
	// make sure to pile up their IDs so we later can skip any already dead
	// containers. This is necessary as while container lifecycle-related events
	// (especially "start" and "die") are properly ordered, fake "start" events
	// from listing containers aren't synchronized and properly ordered with
	// respect to the event log.
	ww.eventgate.Lock()
	ww.ongoinglisting = true
	ww.eventgate.Unlock()
	// Now try to scan the currently available containers and take only the
	// alive into further consideration. This is a potentially lengthy
	// operation, as we need to inspect each potential candidate individually
	// due to the way the Docker daemon's API is designed.
	containers, err := ww.moby.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return // list? what list??
	}
	alives := make([]*Container, 0, len(containers))
	for _, container := range containers {
		if alive, err := newContainer(ctx, ww.moby, container.ID); err == nil {
			alives = append(alives, alive)
		}
	}
	// We now lock out any competing container demise events so we can update
	// the portfolio from the list scan results atomically, but still taking
	// into account all those containers that have gone in the time frame where
	// we scanned for alive containers.
	ww.eventgate.Lock()
	defer func() {
		ww.bluenorwegians = []string{}
		ww.ongoinglisting = false // not strictly necessary here, but anywhere within the gated zone.
		ww.eventgate.Unlock()
	}()
	ww.pfmux.RLock()
	pf := ww.writeportfolio
	ww.pfmux.RUnlock()
nextpet:
	for _, alive := range alives {
		// Did the container die inbetween...? Then skip it and get another pet.
		for _, bluenorwegian := range ww.bluenorwegians {
			if bluenorwegian == alive.ID {
				continue nextpet
			}
		}
		// Otherwise, add it to our portfolio; this is "quick" operation without
		// any trips to the container engine (we already did the "slow" and
		// time-consuming bits before).
		pf.add(alive)
	}
	// Clear list of dead parrots and carry on.
}
