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
	"errors"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
)

var ErrEventStreamStopped = errors.New("event stream stopped")

// Events returns a stream of fake events. It ignores all options, but checks
// ctx for being Done (with or without any error) and then mirrors the context
// error to the (events) error channel returned by Events. After an error the
// event channel will be closed automatically.
//
// Please note that only a single call to the Events API method is supported per
// mock client instance.
func (mm *MockingMoby) Events(ctx context.Context, options types.EventsOptions) (<-chan events.Message, <-chan error) {
	events := make(chan events.Message, 10)
	errs := make(chan error, 1)
	abort := make(chan error, 1)
	mm.emux.Lock()
	mm.events = events
	mm.errs = errs
	mm.abort = abort
	mm.emux.Unlock()
	// Wait in the background for the context to become (well?) done, then
	// propagate any context error to our event error channel and finally be
	// done with it all.
	go func() {
		defer close(errs)
		select {
		case <-ctx.Done():
			errs <- ctx.Err()
		case err := <-abort:
			errs <- err
		}
		mm.emux.Lock()
		defer mm.emux.Unlock()
		mm.events = nil
		mm.errs = nil
		mm.abort = nil
	}()
	return events, errs
}

// StopEvents closes down streaming events with an error on the error channel;
// it is used in unit tests to simulate event stream errors other than a
// cancelled context.
func (mm *MockingMoby) StopEvents() {
	mm.emux.Lock()
	defer mm.emux.Unlock()
	if mm.abort == nil { // ...safeguard against own stupidity
		panic("MockingMoby.StopEvents() without Event()")
	}
	mm.abort <- ErrEventStreamStopped
}

// containerEvent generates a fake container event for the specified action and
// actor.
func (mm *MockingMoby) containerEvent(action string, actor events.Actor) {
	mm.emux.Lock()
	evs := mm.events
	mm.emux.Unlock()
	if evs != nil {
		evs <- events.Message{
			Type:   events.ContainerEventType,
			Action: action,
			Actor:  actor,
			Scope:  "local",
		}
	}
}
