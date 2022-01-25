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

package moby

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/thediveo/whalewatcher"
	"github.com/thediveo/whalewatcher/engineclient"
)

// Type specifies this container engine's type identifier.
const Type = "docker.com"

// ComposerProjectLabel is the name of an optional container label identifying
// the composer project a container is part of.
const ComposerProjectLabel = "com.docker.compose.project"

// PrivilegedLabel is the name of an optional container label signalling be its
// sheer presence that labelled container has been started with a host config
// that includes Privileged. The label's value is always empty, so neither
// "true" nor "false" values here.
const PrivilegedLabel = "github.com/thediveo/whalewatcher/moby/privileged"

// MobyAPIClient is a Docker client offering the container and system APIs. For
// production, Docker's client.Client is a compatible implementation, for unit
// testing our very own mockingmoby.MockingMoby.
type MobyAPIClient interface {
	client.ContainerAPIClient
	client.SystemAPIClient
	DaemonHost() string
	Close() error
}

// MobyWatcher is a Docker-engine EngineClient for interfacing the generic whale
// watching with Docker daemons.
type MobyWatcher struct {
	pid  int           // optional engine PID when known.
	moby MobyAPIClient // (minimal) moby engine API client.
}

// Make sure that the EngineClient interface is fully implemented
var _ (engineclient.EngineClient) = (*MobyWatcher)(nil)

// NewMobyWatcher returns a new MobyWatcher using the specified Docker engine
// client; typically, you would want to use this lower-level constructor only in
// unit tests and instead use watcher.moby.New instead in most use cases.
func NewMobyWatcher(moby MobyAPIClient, opts ...NewOption) *MobyWatcher {
	mw := &MobyWatcher{
		moby: moby,
	}
	for _, opt := range opts {
		opt(mw)
	}
	return mw
}

// NewOption represents options to NewMobyWatcher when creating new watchers
// keeping eyes on moby engines.
type NewOption func(*MobyWatcher)

// WithPID sets the engine's PID when known.
func WithPID(pid int) NewOption {
	return func(mw *MobyWatcher) {
		mw.pid = pid
	}
}

// ID returns the (more or less) unique engine identifier; the exact format is
// engine-specific.
func (mw *MobyWatcher) ID(ctx context.Context) string {
	info, err := mw.moby.Info(ctx)
	if err == nil {
		return info.ID
	}
	return ""
}

// Type returns the type identifier for this container engine.
func (mw *MobyWatcher) Type() string { return Type }

// API returns the container engine API path.
func (mw *MobyWatcher) API() string { return mw.moby.DaemonHost() }

// PID returns the container engine PID, when known.
func (mw *MobyWatcher) PID() int { return mw.pid }

// Client returns the underlying engine client (engine-specific).
func (mw *MobyWatcher) Client() interface{} { return mw.moby }

// Close cleans up and release any engine client resources, if necessary.
func (mw *MobyWatcher) Close() {
	mw.moby.Close()
}

// List all the currently alive and kicking containers, but do not list any
// containers without any processes.
func (mw *MobyWatcher) List(ctx context.Context) ([]*whalewatcher.Container, error) {
	// Scan the currently available containers and take only the alive into
	// further consideration. This is a potentially lengthy operation, as we
	// need to inspect each potential candidate individually due to the way the
	// Docker daemon's API is designed.
	containers, err := mw.moby.ContainerList(ctx, types.ContainerListOptions{})
	if err != nil {
		return nil, err // list? what list??
	}
	alives := make([]*whalewatcher.Container, 0, len(containers))
	for _, container := range containers {
		if alive, err := mw.Inspect(ctx, container.ID); err == nil {
			alives = append(alives, alive)
		} else {
			// silently ignore missing containers that have gone since the list
			// was prepared, but abort on severe problems in order to not keep
			// this running for too long unnecessarily.
			if !engineclient.IsProcesslessContainer(err) && !client.IsErrNotFound(err) {
				return nil, err
			}
		}
	}
	return alives, nil
}

// Inspect (only) those container details of interest to us, given the name or
// ID of a container.
func (mw *MobyWatcher) Inspect(ctx context.Context, nameorid string) (*whalewatcher.Container, error) {
	details, err := mw.moby.ContainerInspect(ctx, nameorid)
	if err != nil {
		return nil, err
	}
	if details.State == nil || details.State.Pid == 0 {
		return nil, engineclient.NewProcesslessContainerError(nameorid, "Docker")
	}
	cntr := &whalewatcher.Container{
		ID:      details.ID,
		Name:    details.Name[1:], // get rid off the leading slash
		Labels:  details.Config.Labels,
		PID:     details.State.Pid,
		Project: details.Config.Labels[ComposerProjectLabel],
		Paused:  details.State.Paused,
	}
	if details.HostConfig != nil && details.HostConfig.Privileged {
		// Just the presence of the "magic" label is sufficient; the label's
		// value doesn't matter.
		cntr.Labels[PrivilegedLabel] = ""
	}
	return cntr, nil
}

// LifecycleEvents streams container engine events, limited just to those events
// in the lifecycle of containers getting born (=alive, as opposed to, say,
// "conceived") and die.
func (mw *MobyWatcher) LifecycleEvents(ctx context.Context) (<-chan engineclient.ContainerEvent, <-chan error) {
	cntreventstream := make(chan engineclient.ContainerEvent)
	cntrerrstream := make(chan error, 1)

	go func() {
		defer close(cntrerrstream)
		evfilters := filters.NewArgs(
			filters.KeyValuePair{Key: "type", Value: "container"},
			filters.KeyValuePair{Key: "event", Value: "start"},
			filters.KeyValuePair{Key: "event", Value: "die"},
			filters.KeyValuePair{Key: "event", Value: "pause"},
			filters.KeyValuePair{Key: "event", Value: "unpause"},
		)
		evs, errs := mw.moby.Events(ctx, types.EventsOptions{Filters: evfilters})
		for {
			select {
			case err := <-errs:
				// The reason of a cancelled context has been flattened into the
				// client's event stream error, grrr. We thus first check on a
				// cancelled context in case of any event stream error and let
				// that take priority.
				if ctx.Err() == context.Canceled {
					err = ctx.Err()
				}
				cntrerrstream <- err
				return
			case ev := <-evs:
				switch ev.Action {
				case "start":
					cntreventstream <- engineclient.ContainerEvent{
						Type:    engineclient.ContainerStarted,
						ID:      ev.Actor.ID,
						Project: ev.Actor.Attributes[ComposerProjectLabel],
					}
				case "die":
					cntreventstream <- engineclient.ContainerEvent{
						Type:    engineclient.ContainerExited,
						ID:      ev.Actor.ID,
						Project: ev.Actor.Attributes[ComposerProjectLabel],
					}
				case "pause":
					cntreventstream <- engineclient.ContainerEvent{
						Type:    engineclient.ContainerPaused,
						ID:      ev.Actor.ID,
						Project: ev.Actor.Attributes[ComposerProjectLabel],
					}
				case "unpause":
					cntreventstream <- engineclient.ContainerEvent{
						Type:    engineclient.ContainerUnpaused,
						ID:      ev.Actor.ID,
						Project: ev.Actor.Attributes[ComposerProjectLabel],
					}
				}
			}
		}
	}()

	return cntreventstream, cntrerrstream
}
