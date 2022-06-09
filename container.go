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
	"fmt"
)

// Container is a deliberately limited view on containers, dealing with only
// those few bits of data we're interested in for watching alive containers and
// how they optionally are organized into composer projects.
//
// Please note that Container objects are considered immutable, so there's no
// locking them for updating.
//
// We consider containers to be alive when they have an initial process (which
// might be frozen) and thus a PID corresponding with that initial process. In
// contrast, we don't care about containers which are either dead without any
// container process(es) or are just yet created and thus still without any
// container process(es).
type Container struct {
	ID       string            // unique identifier of this container.
	Name     string            // user-friendly name of this container.
	Labels   map[string]string // labels assigned to this container.
	PID      int               // PID of container's initial ("ealdorman") process.
	Project  string            // optional composer project name, or zero.
	Paused   bool              // true if container is paused, false if running.
	Rucksack interface{}       // optional additional application-specific container information.
}

// ProjectName returns the name of the composer project for this container, if
// any; otherwise, it returns "" if a container isn't associated with a composer
// project.
func (c Container) ProjectName() string {
	return c.Project
}

// String renders a textual representation of the information kept about a
// specific container, such as its name, ID, and PID.
func (c Container) String() string {
	var pinfo string
	proj := c.ProjectName()
	if proj != "" {
		pinfo = fmt.Sprintf("from project '%s' ", proj)
	}
	paused := ""
	if c.Paused {
		paused = "paused "
	}
	id := ""
	if c.Name != c.ID {
		id = "/" + c.ID
	}
	return fmt.Sprintf("%scontainer '%s'%s %swith PID %d", paused, c.Name, id, pinfo, c.PID)
}
