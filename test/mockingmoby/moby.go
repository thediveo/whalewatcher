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

package mockingmoby

import (
	"context"
	"sync"

	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
)

// MockingMoby is a mock Docker client implementing only listing all containers,
// inspecting them (limited information only) and receiving container-related
// events. All other service API methods will return a not-implemented error
// when tried.
//
// Please note that only a single call to the Events API method is supported per
// mock client instance.
type MockingMoby struct {
	client.ContainerAPIClient
	client.SystemAPIClient

	mux        sync.RWMutex
	containers map[string]MockedContainer // mocked containers by ID
	names      map[string]string          // maps names to IDs

	emux   sync.Mutex
	events chan events.Message // stream events
	errs   chan error          // signal error
	abort  chan error          // test-controlled abort of event stream
}

// Ensure that all needed service API methods have been implemented.
var (
	_ client.ContainerAPIClient = (*MockingMoby)(nil)
	_ client.SystemAPIClient    = (*MockingMoby)(nil)
)

// NewMockingMoby returns a new instance of a mock Docker client.
func NewMockingMoby() *MockingMoby {
	return &MockingMoby{
		containers: map[string]MockedContainer{},
		names:      map[string]string{},
	}
}

// NegotiateAPIVersion is a mock no-op.
func (mm *MockingMoby) NegotiateAPIVersion(ctx context.Context) {}

// DaemonHost returns the host address used by the client
func (mm *MockingMoby) DaemonHost() string { return "mock://mocked" }

// Close closes the mock client, releasing its internal resources.
func (mm *MockingMoby) Close() error {
	return nil
}

// isCtxCancelled returns an error if the specified Context is done, either
// having been cancelled our reached its deadline. Otherwise, returns nil.
func isCtxCancelled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// AddContainer adds a mocked container and optionally emits a container event
// if the container is in running or paused states.
func (mm *MockingMoby) AddContainer(c MockedContainer) {
	mm.mux.Lock()
	defer mm.mux.Unlock()
	mm.containers[c.ID] = c
	mm.names[c.Name] = c.ID
	switch c.Status {
	case MockedRunning, MockedPaused:
		mm.containerEvent("start", events.Actor{
			ID:         c.ID,
			Attributes: MockAttributes(c),
		})
	}
}

// StopContainer stops a mocked container, but does not remove it yet. It emits
// a container event if the container was in running or paused state.
func (mm *MockingMoby) StopContainer(nameorid string) {
	if c, ok := mm.lookup(nameorid); ok {
		mm.mux.Lock()
		// make sure to emit event only after changing the fake container's
		// state to exited.
		status := c.Status
		c.Status = MockedExited
		c.PID = 0
		mm.containers[c.ID] = c
		mm.mux.Unlock()
		switch status {
		case MockedRunning, MockedPaused:
			mm.containerEvent("die", events.Actor{
				ID:         c.ID,
				Attributes: MockAttributes(c),
			})
		}
	}
}

// RemoveContainer removes a mocked container and emits a container event if the
// container was in running or paused states.
func (mm *MockingMoby) RemoveContainer(nameorid string) {
	if c, ok := mm.lookup(nameorid); ok {
		mm.mux.Lock()
		delete(mm.containers, nameorid)
		delete(mm.names, c.Name)
		mm.mux.Unlock()
		switch c.Status {
		case MockedRunning, MockedPaused:
			mm.containerEvent("die", events.Actor{
				ID:         c.ID,
				Attributes: MockAttributes(c),
			})
		}
	}
}

// PauseContainer pauses a container, if currently running, and emits a
// container pause event.
func (mm *MockingMoby) PauseContainer(nameorid string) {
	if c, ok := mm.lookup(nameorid); ok {
		mm.mux.Lock()
		if c.Status != MockedRunning {
			mm.mux.Unlock()
			return
		}
		c.Status = MockedPaused
		mm.containers[c.ID] = c
		mm.mux.Unlock()
		mm.containerEvent("pause", events.Actor{
			ID:         c.ID,
			Attributes: MockAttributes(c),
		})
	}
}

// UnpauseContainer unpauses a container, if currently paused, and emits a
// container unpause event.
func (mm *MockingMoby) UnpauseContainer(nameorid string) {
	if c, ok := mm.lookup(nameorid); ok {
		mm.mux.Lock()
		if c.Status != MockedPaused {
			mm.mux.Unlock()
			return
		}
		c.Status = MockedRunning
		mm.containers[c.ID] = c
		mm.mux.Unlock()
		mm.containerEvent("unpause", events.Actor{
			ID:         c.ID,
			Attributes: MockAttributes(c),
		})
	}
}

// lookup returns a mocked container identified either by ID or name. If not
// found, returns false.
func (mm *MockingMoby) lookup(nameorid string) (MockedContainer, bool) {
	mm.mux.RLock()
	defer mm.mux.RUnlock()
	c, ok := mm.containers[nameorid]
	if !ok {
		if nameorid, ok = mm.names[nameorid]; ok {
			c, ok = mm.containers[nameorid]
		}
	}
	return c, ok
}

// MockAttributes returns a mocked attributes map for the specified mock
// container, based on the container's labels and additional attributes (namely,
// the container name as opposed to its ID). The attributes map is suitable for
// direct emission in the Actor fields of Docker events.
func MockAttributes(c MockedContainer) map[string]string {
	attrs := map[string]string{}
	for ln, lv := range c.Labels {
		attrs[ln] = lv
	}
	attrs["name"] = c.Name
	return attrs
}
