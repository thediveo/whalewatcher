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
	"sync"
)

// Portfolio represents all known composer projects, including the "zero"
// (unnamed) project. The "zero" project has the zero name and contains all
// containers that are not part of any named composer project. The Portfolio
// manages projects implicitly when adding and removing containers belonging to
// projects. Thus, there is no need to explicitly add or delete composer
// projects.
type Portfolio struct {
	projects map[string]*ComposerProject
	m        sync.RWMutex
}

// NewPortfolio returns a new Portfolio.
func NewPortfolio() *Portfolio {
	pf := &Portfolio{
		projects: make(map[string]*ComposerProject),
	}
	pf.projects[""] = newComposerProject("")
	return pf
}

// Names returns the names of all composer projects sans the "zero" project.
func (pf *Portfolio) Names() []string {
	pf.m.RLock()
	defer pf.m.RUnlock()

	names := make([]string, 0, len(pf.projects)-1)
	for name := range pf.projects {
		if name != "" {
			names = append(names, name)
		}
	}
	return names
}

// Project returns the project with the specified name (including the zero
// project name), or nil if no project with the specified name currently exists.
func (pf *Portfolio) Project(name string) *ComposerProject {
	pf.m.RLock()
	defer pf.m.RUnlock()
	return pf.projects[name]
}

// Container returns the [Container] with the specified name, regardless of
// which project it is in. It returns nil, if no container with the specified
// name could be found.
func (pf *Portfolio) Container(nameorid string) *Container {
	pf.m.RLock()
	defer pf.m.RUnlock()
	for _, project := range pf.projects {
		if container := project.Container(nameorid); container != nil {
			return container
		}
	}
	return nil
}

// ContainerTotal returns the total number of containers over all projects,
// including non-project "standalone" containers.
func (pf *Portfolio) ContainerTotal() (total int) {
	for _, project := range pf.projects {
		total += len(project.containers)
	}
	return
}

// Add a container to the portfolio, creating also its composer project if that
// is not yet known. Returns true if the container was newly added, false if it
// already exists.
func (pf *Portfolio) Add(cntr *Container) bool {
	pf.m.Lock()
	defer pf.m.Unlock()

	// Do we have already the container's project in store or do we need to
	// create it?
	projname := cntr.ProjectName()
	proj, ok := pf.projects[projname]
	if !ok {
		proj = newComposerProject(projname)
		pf.projects[projname] = proj
	}
	// Let the project deal with the gory details of adding or not.
	return proj.add(cntr)
}

// Remove a container identified by its ID or name as well as its composer
// project name from the portfolio, removing its composer project if it was the
// only container left in the project.
//
// The information about the removed container is returned, otherwise if no such
// container exists, nil is returned instead.
func (pf *Portfolio) Remove(nameorid string, project string) (cntr *Container) {
	pf.m.Lock()
	defer pf.m.Unlock()

	if proj, ok := pf.projects[project]; ok {
		cntr = proj.remove(nameorid)
		if project != "" && len(proj.Containers()) == 0 {
			// The (non-zero) project has become empty, so we remove this
			// project from the portfolio.
			delete(pf.projects, project)
		}
	}
	return
}
