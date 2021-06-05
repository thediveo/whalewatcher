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

// MockedContainerStatus is a compressed, only-essentials, no-bulls version of
// Docker's types.ContainerStatus.
type MockedContainerStatus int

// The available states of a mocked container.
const (
	MockedCreated MockedContainerStatus = iota
	MockedRunning
	MockedPaused
	MockedDead
	MockedExited
)

// MockedStates maps the states of a mocked container to Docker's textual
// descriptive (and slightly chatty) container states, suitable for display to
// hoomans.
var MockedStates = map[MockedContainerStatus]string{
	MockedCreated: "",
	MockedRunning: "up for ages",
	MockedPaused:  "pausing a moment",
	MockedDead:    "just sleeping",
	MockedExited:  "exit 42",
}

// MockedStatus maps the states of a mocked container to Docker's container
// status strings that is better suited for code checks (no chatty additions and
// content variations).
var MockedStatus = map[MockedContainerStatus]string{
	MockedCreated: "created",
	MockedRunning: "running",
	MockedPaused:  "paused",
	MockedDead:    "dead",
	MockedExited:  "exited",
}

// MockedContainer is our very, very limited knowledge about a mocked container;
// it just stores the minimum of information we need in mocking our own unit
// tests.
type MockedContainer struct {
	ID     string                // unique identifier of container
	Name   string                // name of container without any prefixing "/"
	Status MockedContainerStatus // container status (without any thrills)
	PID    int                   // PID of initial container process if container is "alive"
	Labels map[string]string     // container labels
}
