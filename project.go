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
	"strings"
	"sync"
)

// ComposerProject represents a set of (running or paused, yet somehow alive)
// containers belonging to a specific Docker Compose/Composer project.
//
// As composer projects are artefacts above the first-level elements of the
// Docker container engine we can only reconstruct them in an extremely limited
// fashion from the live container information available to us. Yet that's fine
// in our context, as we just want to understand the concrete relationships
// between projects and their containers.
type ComposerProject struct {
	Name       string       // composer project name, guaranteed to be constant.
	containers []*Container // containers belonging to this project (unsorted).
	m          sync.RWMutex
}

// newComposerProject returns a new composer project of the specified name and
// without any containers yet.
func newComposerProject(name string) *ComposerProject {
	return &ComposerProject{
		Name:       name,
		containers: []*Container{},
	}
}

// Containers returns the current list of containers in this composer project.
func (p *ComposerProject) Containers() []*Container {
	p.m.RLock()
	defer p.m.RUnlock()
	return p.containers
}

// ContainerNames returns the current list of container names belonging to this
// composer project.
func (p *ComposerProject) ContainerNames() []string {
	p.m.RLock()
	defer p.m.RUnlock()

	names := make([]string, len(p.containers))
	for idx, cntr := range p.containers {
		names[idx] = cntr.Name
	}
	return names
}

// Container returns container information about the container with the
// specified name or ID. If the name or ID wasn't found in this project, then
// nil is returned instead.
func (p *ComposerProject) Container(nameorid string) *Container {
	p.m.RLock()
	defer p.m.RUnlock()

	for _, cntr := range p.containers {
		if cntr.Name == nameorid || cntr.ID == nameorid {
			return cntr
		}
	}
	return nil
}

// SetPaused changes a container's Paused state, obeying the design restriction
// that Container objects are immutable. It returns the container in its new
// state.
func (p *ComposerProject) SetPaused(nameorid string, paused bool) *Container {
	p.m.Lock()
	defer p.m.Unlock()

	for idx, cntr := range p.containers {
		if cntr.Name == nameorid || cntr.ID == nameorid {
			if paused != cntr.Paused {
				// As Container is supposed to be immutable, we clone the existing
				// object, then modify the copy, and finally update the container
				// reference to point to the copy with the updated state.
				c := *cntr
				c.Paused = paused
				p.containers[idx] = &c
				return &c
			}
			return cntr
		}
	}
	// Silently ignore a non-existing name/ID.
	return nil
}

// String returns a textual representation of a composer project with its
// containers (rendering names, but not IDs).
func (p *ComposerProject) String() string {
	p.m.RLock()
	defer p.m.RUnlock()

	if len(p.containers) > 0 {
		cnames := make([]string, 0, len(p.containers))
		for _, cntr := range p.containers {
			cnames = append(cnames, cntr.Name)
		}
		return fmt.Sprintf("composer project '%s' with containers: '%s'",
			p.Name, strings.Join(cnames, "', '"))
	}
	return fmt.Sprintf("empty composer project '%s'", p.Name)
}

// add a new container to a composer project. Silently ignore any attempt to add
// an already existing container. Returns true if the container was newly added,
// false if the container already exists.
func (p *ComposerProject) add(c *Container) bool {
	p.m.Lock()
	defer p.m.Unlock()

	for _, cntr := range p.containers {
		if cntr.Name == c.Name {
			return false
		}
	}
	p.containers = append(p.containers, c)
	return true
}

// remove the container identified by either name or ID from this composer
// project. It returns the information about the removed container, if any, or
// nil.
func (p *ComposerProject) remove(nameid string) *Container {
	p.m.Lock()
	defer p.m.Unlock()

	for idx, cntr := range p.containers {
		if cntr.Name == nameid || cntr.ID == nameid {
			// We've found the container by name or ID, so we new remove it from
			// the slice. As we don't care about order, erm, container order,
			// that is, we do an optimized slice delete, see also:
			// https://github.com/golang/go/wiki/SliceTricks#delete-without-preserving-order
			// Make sure to help the garbage collector by freeing the final
			// slice slot before shortening the slice.
			last := len(p.containers) - 1
			p.containers[idx], p.containers[last] = p.containers[last], nil
			p.containers = p.containers[:last]
			return cntr
		}
	}
	return nil
}
