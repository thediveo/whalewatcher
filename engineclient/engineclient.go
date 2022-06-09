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

package engineclient

import (
	"context"
	"fmt"

	"github.com/thediveo/whalewatcher"
)

// EngineClient defines the generic methods needed in order to watch the
// containers of a container engine, regardless of the specific type of engine.
type EngineClient interface {
	// List all the currently alive and kicking containers. Return errors (only)
	// in case of (severe) conntection and daemon failures that aren't
	// transparent.
	List(ctx context.Context) ([]*whalewatcher.Container, error)
	// Query (only) the subset of container details of interest to us, given the
	// name or ID of a particular container.
	Inspect(ctx context.Context, nameorid string) (*whalewatcher.Container, error)
	// Stream container lifecycle events, limited to those events in the
	// lifecycle of containers getting born (=alive, as opposed to, say,
	// "conceived", dead/gone/"sleeping") and die.
	LifecycleEvents(ctx context.Context) (<-chan ContainerEvent, <-chan error)

	// (More or less) unique engine identifier; the exact format is
	// engine-specific.
	ID(ctx context.Context) string
	// Identifier of the type of container engine, such as "docker.com",
	// "containerd.io", et cetera.
	Type() string
	// Version information about the engine.
	Version(ctx context.Context) string
	// Container engine API path.
	API() string
	// Container engine PID, when known. Otherwise zero.
	PID() int

	// Underlying engine client (engine-specific).
	Client() interface{}

	// Clean up and release any engine client resources, if necessary.
	Close()
}

// RucksackPacker optionally adds additional information to the tracked
// container information, as kind of a Rucksack. It gets passed container
// engine-specific inspection information so as to be able to pick and pack
// application-specific container information beyond the stock information
// always maintained by the whalewatcher module.
type RucksackPacker interface {
	Pack(container *whalewatcher.Container, inspection interface{})
}

// ContainerEventType identifies and enumerates the container lifecycle events
// of "alive" containers, including their demise. Please do not confuse this
// lifecycle for alive containers with the usualy much more comprehensive
// container lifecycles that include creating a container long before it might
// become alive: such stages are of no interest to us here; for instance,
// there's no container creation event, "only" the container start event.
type ContainerEventType byte

const (
	ContainerStarted  ContainerEventType = iota // container has been started
	ContainerExited                             // container has terminated (exited)
	ContainerPaused                             // container has been paused
	ContainerUnpaused                           // container has been unpaused
)

// ProjectUnknown signals that the project name for a container event is
// unknown, as opposed to the zero project name.
const ProjectUnknown = "\000"

// ContainerEvent is either a container lifecycle event of a container becoming
// alive, having died (more precise: its process exited), paused or unpaused.
type ContainerEvent struct {
	Type    ContainerEventType // type of lifecycle event.
	ID      string             // ID (or name) of container.
	Project string             // optional composer project name, or zero.
}

// ErrProcesslessContainer is a custom error indicating that inspecting
// container details failed because it was a container without any process (yet
// or anymore), such as a container freshly created, and yet not started.
//
// This usually happens due to transient changes between listing containers or
// processing lifecycle-related events and inspecting them, because there are no
// atomic operations available (and strictly necessary). There are already
// checks in place that usually avoid this error in static situations, but they
// can never give one hundred percent garantuees.
type ErrProcesslessContainer string

// Error returns the error message.
func (err ErrProcesslessContainer) Error() string {
	return string(err)
}

// NewProcesslessContainerError returns a new ErrProcesslessContainer, stating
// the type of container and its name or ID.
func NewProcesslessContainerError(nameorid string, typ string) error {
	return ErrProcesslessContainer(fmt.Sprintf(
		"%s container '%s' has no initial process", typ, nameorid))
}

// IsProcesslessContainer returns true if the specified error is an
// ProcesslessContainerErr.
func IsProcesslessContainer(err error) bool {
	_, ok := err.(ErrProcesslessContainer)
	return ok
}
