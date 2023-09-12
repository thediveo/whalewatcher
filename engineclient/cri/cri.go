// Copyright 2023 Harald Albrecht.
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

package cri

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/thediveo/whalewatcher"
	"github.com/thediveo/whalewatcher/engineclient"
	runtimev1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// AnnotationKeyPrefix prefixes all Kubernetes annotation keys in order to avoid
// clashes between label keys and annotation keys, because the whalewatcher
// model only defines “labels” as a more generic construct.
const AnnotationKeyPrefix = "annotation.k8s/"

// PodNameLabel specifies the pod name of a container.
const PodNameLabel = "io.kubernetes.pod.name"

// PodNamespaceLabel specifies the namespace of the pod a container is part of.
const PodNamespaceLabel = "io.kubernetes.pod.namespace"

// PodContainerNameLabel specifies the name of a container inside a pod from the
// Kubernetes perspective.
const PodContainerNameLabel = "io.kubernetes.container.name"

// PodUidLabel specifies the UID of a pod (=group).
const PodUidLabel = "io.kubernetes.pod.uid"

// FIXME:
const kubeAPIVersion = "0.1.0"

// Type specifies this container engine's type identifier.
const Type = "k8s.io/cri-api"

// CRIWatcher is a CRI EngineClient for interfacing the generic whale watching
// with container engines that support the CRI API. Oh, it's “CRI”, not
// “Cri”.
type CRIWatcher struct {
	pid    int                         // optional engine PID when known.
	client *Client                     // CRI API client.
	packer engineclient.RucksackPacker // optional Rucksack packer for app-specific container information.
}

// NewCRIWatcher returns a new ContainerdWatcher using the specified
// containerd engine client; normally, you would want to use this lower-level
// constructor only in unit tests.
func NewCRIWatcher(client *Client, opts ...NewOption) *CRIWatcher {
	cw := &CRIWatcher{
		client: client,
	}
	for _, opt := range opts {
		opt(cw)
	}
	return cw
}

// Make sure that the EngineClient interface is fully implemented
var _ (engineclient.EngineClient) = (*CRIWatcher)(nil)

// NewOption represents options to NewCRIWatcher when creating new watchers
// keeping eyes on CRI-supporting container engines.
type NewOption func(*CRIWatcher)

// WithPID sets the engine's PID when known.
func WithPID(pid int) NewOption {
	return func(cw *CRIWatcher) {
		cw.pid = pid
	}
}

// WithRucksackPacker sets the Rucksack packer that adds application-specific
// container information based on the inspected container data. The specified
// Rucksack packer gets passed the inspection data in form of a
// InspectionDetails.
func WithRucksackPacker(packer engineclient.RucksackPacker) NewOption {
	return func(cw *CRIWatcher) {
		cw.packer = packer
	}
}

// ID returns the (more or less) unique engine identifier; the exact format is
// engine-specific.
func (cw *CRIWatcher) ID(ctx context.Context) string {
	// CRI doesn't (directly) support container engine identifications.
	return ""
}

// Type returns the type identifier for this container engine.
func (cw *CRIWatcher) Type() string { return Type }

// Version information about the engine.
func (cw *CRIWatcher) Version(ctx context.Context) string {
	version, err := cw.client.rtcl.Version(ctx, &runtimev1.VersionRequest{
		Version: kubeAPIVersion,
	})
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%s %s [API %s]",
		version.RuntimeName, version.RuntimeVersion, version.RuntimeApiVersion)
}

// API returns the container engine API path.
func (cw *CRIWatcher) API() string { return cw.client.conn.Target() }

// PID returns the container engine PID, when known.
func (cw *CRIWatcher) PID() int { return cw.pid }

// Client returns the underlying engine client (engine-specific).
func (cw *CRIWatcher) Client() interface{} { return cw.client }

// Close cleans up and release any engine client resources, if necessary.
func (cw *CRIWatcher) Close() {
	cw.client.conn.Close()
}

// List all the currently alive and kicking containers. In case of the CRI API
// this actually turns out to be a someone involved process, as the API has been
// designed solely from the kubelet perspective and thus tends to becomde
// inefficient in other use cases.
func (cw *CRIWatcher) List(ctx context.Context) ([]*whalewatcher.Container, error) {
	cntrs, err := cw.client.rtcl.ListContainers(ctx, &runtimev1.ListContainersRequest{
		Filter: &runtimev1.ContainerFilter{
			State: &runtimev1.ContainerStateValue{State: runtimev1.ContainerState_CONTAINER_RUNNING},
		},
	})
	if err != nil {
		return nil, err
	}
	containers := []*whalewatcher.Container{}
	for _, cntr := range cntrs.Containers {
		if cntr.State != runtimev1.ContainerState_CONTAINER_RUNNING {
			continue
		}
		wwcntr := cw.newContainer(ctx, cntr, nil)
		containers = append(containers, wwcntr)
	}
	return containers, nil
}

// Inspect (only) those container details of interest to us, given the name or
// ID of a container.
func (cw *CRIWatcher) Inspect(ctx context.Context, nameorid string) (*whalewatcher.Container, error) {
	cntrs, err := cw.client.rtcl.ListContainers(ctx, &runtimev1.ListContainersRequest{
		Filter: &runtimev1.ContainerFilter{Id: nameorid},
	})
	if err != nil {
		return nil, err
	}
	if len(cntrs.Containers) != 1 {
		return nil, fmt.Errorf("cannot inspect container with id %q", nameorid)
	}
	return cw.newContainer(ctx, cntrs.Containers[0], nil), nil
}

// newContainer returns the container details of interest to us. If the container is
// not alive (with a process), then nil is returned instead.
func (cw *CRIWatcher) newContainer(
	ctx context.Context,
	cntr *runtimev1.Container,
	optPod *runtimev1.PodSandbox,
) *whalewatcher.Container {
	if cntr.State != runtimev1.ContainerState_CONTAINER_RUNNING {
		return nil
	}
	// If we didn't get the related pod details, then we need to query them now.
	pods, err := cw.client.rtcl.ListPodSandbox(ctx, &runtimev1.ListPodSandboxRequest{
		Filter: &runtimev1.PodSandboxFilter{Id: cntr.PodSandboxId}})
	if err != nil || len(pods.Items) != 1 {
		return nil
	}
	// We still don't know this container's PID and the CRI API actually
	// doesn't provide it anywhere. Instead, at least some CRI-providing
	// container engines reveal container PIDs through the "info" element of
	// the container status. Well, another round trip to the container
	// engine, then. Thanks CRI for nothing.
	status, err := cw.client.rtcl.ContainerStatus(ctx, &runtimev1.ContainerStatusRequest{
		ContainerId: cntr.Id,
		Verbose:     true,
	})
	if err != nil {
		return nil
	}
	info := status.Info["info"]
	if info == "" {
		return nil
	}
	var innerInfo struct {
		PID int `json:"pid"`
	}
	if err := json.Unmarshal([]byte(info), &innerInfo); err != nil {
		return nil
	}
	// Map annotations to the generic labels, using a unique key prefix to make
	// them easily and determistically detectable.
	labels := cntr.Labels
	for key, value := range cntr.Annotations {
		labels[AnnotationKeyPrefix+key] = value
	}
	labels[PodUidLabel] = pods.Items[0].Metadata.Uid
	labels[PodNameLabel] = pods.Items[0].Metadata.Name
	labels[PodNamespaceLabel] = pods.Items[0].Metadata.Namespace
	labels[PodContainerNameLabel] = cntr.Metadata.Name

	return &whalewatcher.Container{
		ID:     cntr.Id,
		Name:   cntr.Metadata.Name,
		Labels: labels,
		PID:    innerInfo.PID,
		Paused: false, // there is no pause notion in Kubernetes
	}
}

// LifecycleEvents streams container engine events, limited just to those events
// in the lifecycle of containers getting born (=alive, as opposed to, say,
// "conceived") and die.
func (cw *CRIWatcher) LifecycleEvents(ctx context.Context) (
	<-chan engineclient.ContainerEvent, <-chan error,
) {
	cntreventstream := make(chan engineclient.ContainerEvent, 16)
	cntrerrstream := make(chan error, 1)

	go func() {
		defer close(cntrerrstream)
		evcl, err := cw.client.rtcl.GetContainerEvents(ctx,
			&runtimev1.GetEventsRequest{ /*nothing*/ })
		if err != nil {
			cntrerrstream <- err
			return
		}
		for {
			// Recv will properly return a cancelled context error when the
			// context is cancelled that we specified in the call to
			// GetContainerEvents.
			// https://github.com/containerd/containerd/blob/4c538164e60eb8425914c353db783afd62c1bc79/integration/container_event_test.go#L108
			ev, err := evcl.Recv()
			if err != nil {
				if ctx.Err() == context.Canceled {
					err = ctx.Err()
				}
				cntrerrstream <- err
				return
			}
			// At least in the case of containerd, the sandbox lifcycle also
			// emits container events with their ContainerId equal to the
			// PodSandboxStatus.Id. Please see also:
			// https://github.com/containerd/containerd/blob/4d2c8879908285454a4006534cb0af82bb58a406/pkg/cri/server/sandbox_run.go#L506
			switch ev.ContainerEventType {
			case runtimev1.ContainerEventType_CONTAINER_STARTED_EVENT:
				cntreventstream <- engineclient.ContainerEvent{
					Type: engineclient.ContainerStarted,
					ID:   ev.ContainerId,
				}
			case runtimev1.ContainerEventType_CONTAINER_STOPPED_EVENT:
				cntreventstream <- engineclient.ContainerEvent{
					Type: engineclient.ContainerExited,
					ID:   ev.ContainerId,
				}
			}
		}
	}()

	return cntreventstream, cntrerrstream
}
