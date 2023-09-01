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

package containerd

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/api/types/task"
	"github.com/containerd/containerd/containers"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/typeurl/v2"
	"github.com/thediveo/whalewatcher"
	"github.com/thediveo/whalewatcher/engineclient"
)

// Type specifies this container engine's type identifier.
const Type = "containerd.io"

// DockerNamespace is the name of the containerd namespace used by Docker for
// its own containers (and tasks). As the whalewatcher module has a dedicated
// Docker engine client, we need to skip this namespace -- the rationale is that
// especially the container name is missing at the containerd engine level, but
// only available via the docker/moby API.
const DockerNamespace = "moby"

// ComposerProjectLabel is the name of an optional container label identifying
// the composer project a container is part of. We don't import the definition
// from the moby package in order to not having to rely on that dependency.
const ComposerProjectLabel = "com.docker.compose.project"

// NerdctlLabelPrefix is the label key prefix used to namespace (oh no, not
// another "namespace") nertctl-related labels. On purpose, we don't import
// nerdctl's definitions in order to avoid even more dependencies.
const NerdctlLabelPrefix = "nerdctl/"

// NerdctlNameLabel stores a container's name, as opposed to the ID.
const NerdctlNameLabel = NerdctlLabelPrefix + "name"

// nsdelemiter is the delemiter used to separate a containerd namespace from a
// containerd ID.
const nsdelemiter = "/"

// ContainerdWatcher is a containerd EngineClient for interfacing the generic
// whale watching with containerd daemons.
type ContainerdWatcher struct {
	pid    int                         // optional engine PID when known.
	client *containerd.Client          // containerd API client.
	packer engineclient.RucksackPacker // optional Rucksack packer for app-specific container information.
}

// NewContainerdWatcher returns a new ContainerdWatcher using the specified
// containerd engine client; normally, you would want to use this lower-level
// constructor only in unit tests.
func NewContainerdWatcher(client *containerd.Client, opts ...NewOption) *ContainerdWatcher {
	cw := &ContainerdWatcher{
		client: client,
	}
	for _, opt := range opts {
		opt(cw)
	}
	return cw
}

// Make sure that the EngineClient interface is fully implemented
var _ (engineclient.EngineClient) = (*ContainerdWatcher)(nil)

// NewOption represents options to NewContainerdWatcher when creating new
// watchers keeping eyes on containerd engines.
type NewOption func(*ContainerdWatcher)

// WithPID sets the engine's PID when known.
func WithPID(pid int) NewOption {
	return func(cw *ContainerdWatcher) {
		cw.pid = pid
	}
}

// WithRucksackPacker sets the Rucksack packer that adds application-specific
// container information based on the inspected container data. The specified
// Rucksack packer gets passed the inspection data in form of a
// InspectionDetails.
func WithRucksackPacker(packer engineclient.RucksackPacker) NewOption {
	return func(cw *ContainerdWatcher) {
		cw.packer = packer
	}
}

// InspectionDetails combines the container inspection details with its task
// process details. InspectionDetails gets passed to Rucksack packers where
// registered in order to pick additional inspection information beyond the
// generic staple data maintained by the whalewatcher module.
type InspectionDetails struct {
	*containers.Container
	*task.Process
}

// ID returns the (more or less) unique engine identifier; the exact format is
// engine-specific.
func (cw *ContainerdWatcher) ID(ctx context.Context) string {
	serverinfo, err := cw.client.Server(ctx)
	if err != nil {
		// Older containerd versions before 1.3(?) don't support the server
		// information API.
		return ""
	}
	return serverinfo.UUID
}

// Type returns the type identifier for this container engine.
func (cw *ContainerdWatcher) Type() string { return Type }

// Version information about the engine.
func (cw *ContainerdWatcher) Version(ctx context.Context) string {
	version, err := cw.client.Version(ctx)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s", version.Version, version.Revision)
}

// API returns the container engine API path.
func (cw *ContainerdWatcher) API() string { return cw.client.Conn().Target() }

// PID returns the container engine PID, when known.
func (cw *ContainerdWatcher) PID() int { return cw.pid }

// Client returns the underlying engine client (engine-specific).
func (cw *ContainerdWatcher) Client() interface{} { return cw.client }

// Close cleans up and release any engine client resources, if necessary.
func (cw *ContainerdWatcher) Close() {
	cw.client.Close()
}

// List all the currently alive and kicking containers, but do not list any
// containers without any processes.
func (cw *ContainerdWatcher) List(ctx context.Context) ([]*whalewatcher.Container, error) {
	// As containerd organizes containers (and tasks) into so-called
	// "spaces" (argh, yet another kind of "namespace"!) we first need to
	// iterate them all.
	spaces, err := cw.client.NamespaceService().List(ctx)
	if err != nil {
		return nil, err
	}
	// And now for the details...
	containerAPI := cw.client.ContainerService()
	taskAPI := cw.client.TaskService()
	containers := []*whalewatcher.Container{}
	for _, namespace := range spaces {
		// Skip Docker's/moby's namespace, as this is managed by the Docker
		// daemon and we cannot discover all relevant container information at
		// the containerd level; namely, the container name (as opposed to its
		// ID) is missing.
		if namespace == DockerNamespace {
			continue
		}
		// Prepare namespace'd context for further API calls and then get the
		// container details.
		nsctx := namespaces.WithNamespace(ctx, namespace)
		// As labels are considered to be a container's configuration as opposed
		// to a container's state information, we first have to list all
		// containers and then index their labels.
		cntrs, err := containerAPI.List(nsctx)
		if err != nil {
			continue // silently skip this namespace
		}
		cntrlabels := map[string]map[string]string{}
		for _, container := range cntrs {
			cntrlabels[container.ID] = container.Labels
		}
		// Only now can we look for signs of container life...
		tasks, err := taskAPI.List(nsctx, &tasks.ListTasksRequest{})
		if err != nil {
			continue // silently skip this namespace
		}
		for _, task := range tasks.Tasks {
			cntr := cw.newContainer(namespace, cntrlabels[task.ID], task)
			if cntr == nil {
				continue
			}
			containers = append(containers, cntr)
		}
	}
	return containers, nil
}

// Inspect (only) those container details of interest to us, given the name or
// ID of a container.
func (cw *ContainerdWatcher) Inspect(ctx context.Context, nameorid string) (*whalewatcher.Container, error) {
	namespace, id := decodeDisplayID(nameorid)
	nsctx := namespaces.WithNamespace(ctx, namespace)
	cntr, err := cw.client.ContainerService().Get(nsctx, id)
	if err != nil {
		return nil, err
	}
	task, err := cw.client.TaskService().Get(nsctx, &tasks.GetRequest{ContainerID: id})
	if err != nil {
		return nil, err
	}
	c := cw.newContainer(namespace, cntr.Labels, task.Process)
	if c == nil {
		return nil, engineclient.NewProcesslessContainerError(nameorid, "containerd")
	}
	if cw.packer != nil {
		cw.packer.Pack(c, InspectionDetails{
			Container: &cntr,
			Process:   task.Process,
		})
	}
	return c, nil
}

// newContainer returns the container details of interest to us, given a task,
// namespace, and container labels. If the task is not alive (with a process),
// then nil is returned instead.
//
// Note: since containerd features "namespaces", we have to namespace the ID, by
// prepending the namespace to the ID in case its not the "default" namespace.
func (cw *ContainerdWatcher) newContainer(namespace string, labels map[string]string, proc *task.Process) *whalewatcher.Container {
	paused := false
	switch proc.Status {
	case task.Status_RUNNING:
		break
	case task.Status_PAUSING, task.Status_PAUSED:
		paused = true
	default:
		return nil
	}
	// While containerd itself unfortunately doesn't follow Docker's concept of
	// differentiating between an always container instance-unique ID versus a
	// functional name, nerdctl emulates it using a nerdctl-specific container
	// label. If that's present, then we'll happily use it. Slightly differing
	// from Docker, any name will always be prefixed by a (non-default)
	// namespace.
	id := displayID(namespace, proc.ID)
	name := id
	if nerdyname, ok := labels[NerdctlNameLabel]; ok {
		name = displayID(namespace, nerdyname)
	}
	// nerdctl now supports the composer project label.
	projectname := labels[ComposerProjectLabel]
	return &whalewatcher.Container{
		ID:      id,
		Name:    name,
		Project: projectname,
		PID:     int(proc.Pid),
		Labels:  labels,
		Paused:  paused,
	}
}

// LifecycleEvents streams container engine events, limited just to those events
// in the lifecycle of containers getting born (=alive, as opposed to, say,
// "conceived") and die.
func (cw *ContainerdWatcher) LifecycleEvents(ctx context.Context) (
	<-chan engineclient.ContainerEvent, <-chan error,
) {
	cntreventstream := make(chan engineclient.ContainerEvent)
	cntrerrstream := make(chan error, 1)

	go func() {
		defer close(cntrerrstream)
		evs, errs := cw.client.EventService().Subscribe(ctx,
			// please note that strings need to be enclosed in quotes, otherwise
			// silent fail...
			`topic=="/tasks/start"`,
			`topic=="/tasks/exit"`,
			`topic=="/tasks/paused"`,
			`topic=="/tasks/resumed"`)
		for {
			select {
			case err := <-errs:
				if ctx.Err() == context.Canceled {
					err = ctx.Err()
				}
				cntrerrstream <- err
				return
			case ev := <-evs:
				// We here ignore Docker's containerd namespace, as "genuine"
				// Docker containers must be handled at the level of the Docker
				// daemon (API) instead. The reason is that there's no Docker
				// container name at the containerd level, only the container
				// ID.
				if ev.Namespace == DockerNamespace {
					continue
				}
				details, err := typeurl.UnmarshalAny(ev.Event)
				if err != nil {
					continue
				}
				// Unfortunately, containerd engine events differ from Docker
				// engine events in that the task start/stop events do not carry
				// any container labels. and especially not a composer project
				// label.
				switch ev.Topic {
				case "/tasks/start":
					taskstart := details.(*events.TaskStart)
					cntreventstream <- engineclient.ContainerEvent{
						Type:    engineclient.ContainerStarted,
						ID:      displayID(ev.Namespace, taskstart.ContainerID),
						Project: engineclient.ProjectUnknown,
					}
				case "/tasks/exit":
					taskexit := details.(*events.TaskExit)
					cntreventstream <- engineclient.ContainerEvent{
						Type:    engineclient.ContainerExited,
						ID:      displayID(ev.Namespace, taskexit.ContainerID),
						Project: engineclient.ProjectUnknown,
					}
				case "/tasks/paused":
					taskpaused := details.(*events.TaskPaused)
					cntreventstream <- engineclient.ContainerEvent{
						Type:    engineclient.ContainerPaused,
						ID:      displayID(ev.Namespace, taskpaused.ContainerID),
						Project: engineclient.ProjectUnknown,
					}
				case "/tasks/resumed":
					taskresumed := details.(*events.TaskResumed)
					cntreventstream <- engineclient.ContainerEvent{
						Type:    engineclient.ContainerUnpaused,
						ID:      displayID(ev.Namespace, taskresumed.ContainerID),
						Project: engineclient.ProjectUnknown,
					}
				}
			}
		}
	}()

	return cntreventstream, cntrerrstream
}

// displayID takes a containerd namespace and container ID and returns a
// displayable ID for it.
func displayID(namespace, id string) string {
	if namespace == "default" {
		return id
	}
	return namespace + nsdelemiter + id
}

// decodeDisplayID splits a displayable ID into its containerd namespace and
// container ID elements.
func decodeDisplayID(displayid string) (namespace, id string) {
	parts := strings.SplitN(displayid, nsdelemiter, 2)
	if len(parts) < 2 {
		return "default", displayid
	}
	return parts[0], parts[1]
}
