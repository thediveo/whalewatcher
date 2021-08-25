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
	// Container engine API path.
	API() string
	// Container engine PID, when known. Otherwise zero.
	PID() int

	// Underlying engine client (engine-specific).
	Client() interface{}

	// Clean up and release any engine client resources, if necessary.
	Close()
}

// ContainerEventType identifies and enumerates the (few) container lifecycle
// events we're interested in, regardless of a particular container engine.
type ContainerEventType byte

// Container lifecycle events, covering only "alive" containers.
const (
	ContainerStarted ContainerEventType = iota
	ContainerExited
	ContainerPaused
	ContainerUnpaused
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
