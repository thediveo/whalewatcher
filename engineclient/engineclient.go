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
	// List for the currently alive and kicking containers.
	List(ctx context.Context) ([]*whalewatcher.Container, error)
	// Query (only) those container details of interest to us, given the name or
	// ID of a container.
	Inspect(ctx context.Context, nameorid string) (*whalewatcher.Container, error)
	// Stream container lifecycle events, limited to those events in the
	// lifecycle of containers getting born (=alive, as opposed to, say,
	// "conceived") and die.
	LifecycleEvents(ctx context.Context) (<-chan ContainerEvent, <-chan error)

	// (More or less) unique engine identifier; the exact format is
	// engine-specific.
	ID(ctx context.Context) string

	// Clean up and release any engine client resources, if necessary.
	Close()
}

// ContainerEvent is either a container lifecycle event of a container becoming
// alive or having died.
type ContainerEvent struct {
	Born    bool   // true if new container, false if container has died.
	ID      string // ID (or name) of container.
	Project string // optional composer project name, or zero.
}
