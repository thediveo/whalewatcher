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
	"golang.org/x/exp/maps"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// AnnotationKeyPrefix prefixes all Kubernetes annotation keys in order to avoid
// clashes between label keys and annotation keys, because the whalewatcher
// model only defines “labels” as a more generic construct. And since we're here
// on the whalewatcher/lxkns level, the Kubernetes rules for label and annotation
// keys don't applay anymore.
const AnnotationKeyPrefix = "io.kubernetes.annotation/"

// PodNameLabel specifies the pod name of a container.
const PodNameLabel = "io.kubernetes.pod.name"

// PodNamespaceLabel specifies the namespace of the pod a container (or the pod
// sandbox) is part of.
const PodNamespaceLabel = "io.kubernetes.pod.namespace"

// PodContainerNameLabel specifies the name of a container inside a pod from the
// Kubernetes perspective.
const PodContainerNameLabel = "io.kubernetes.container.name"

// PodUidLabel specifies the UID of a pod (=group).
const PodUidLabel = "io.kubernetes.pod.uid"

// PodSandboxLabel marks a container as the pod sandbox (or “pause”) container;
// this label is present only on sandbox containers and the label value is
// irrelevant.
const PodSandboxLabel = "io.kubernetes.sandbox"

// We use kubeAPIVersion to identify us to the CRI API provider; now, this is
// flaky territory in the Evented PLEG API: is isn't (yet) checked in any way
// and there is no specification as to what exactly needs to be specified here.
// Is it the API semver with or without the “v” prefix? Is it the 1.x semver of
// the API specification, or is it the semver of the Go API implementation...?
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
// Rucksack packer gets passed the inspection data in form of
// InspectionDetails.
func WithRucksackPacker(packer engineclient.RucksackPacker) NewOption {
	return func(cw *CRIWatcher) {
		cw.packer = packer
	}
}

// ID returns the (more or less) unique engine identifier; the exact format is
// engine-specific. Unfortunately, the CRI API doesn't has any concept or notion
// of individual “engine identification”. We thus synthesize one from the host
// name, going down the rabit hole of UTS and mount namespaces...
func (cw *CRIWatcher) ID(ctx context.Context) string {
	// CRI doesn't (directly) support container engine identifications.
	return hostname(cw.pid)
}

// Type returns the type identifier for this container engine.
func (cw *CRIWatcher) Type() string { return Type }

// Version information about the engine.
func (cw *CRIWatcher) Version(ctx context.Context) string {
	version, err := cw.client.rtcl.Version(ctx, &runtime.VersionRequest{
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

// List all the currently alive and kicking containers (including pod sandboxes,
// which are also containers).
//
// In case of the CRI API this actually turns out to be a somewhat involved
// process, as the API has been designed solely from the kubelet perspective and
// thus tends to become unwieldly in other use cases.
func (cw *CRIWatcher) List(ctx context.Context) ([]*whalewatcher.Container, error) {
	// List all containers and find out which pods they belong to; this won't
	// give us the sandbox containers though...
	cntrs, err := cw.client.rtcl.ListContainers(ctx, &runtime.ListContainersRequest{
		Filter: &runtime.ContainerFilter{
			State: &runtime.ContainerStateValue{State: runtime.ContainerState_CONTAINER_RUNNING},
		},
	})
	if err != nil {
		return nil, err
	}
	containers := []*whalewatcher.Container{}
	for _, cntr := range cntrs.Containers {
		wwcntr := cw.newContainer(ctx, cntr, nil)
		if wwcntr == nil {
			continue
		}
		containers = append(containers, wwcntr)
	}
	// Now additionally list the sandbox containers and create container
	// information for them too.
	sandboxes, err := cw.client.rtcl.ListPodSandbox(ctx, &runtime.ListPodSandboxRequest{
		Filter: &runtime.PodSandboxFilter{
			State: &runtime.PodSandboxStateValue{State: runtime.PodSandboxState_SANDBOX_READY},
		},
	})
	if err != nil {
		return nil, err
	}
	for _, sandbox := range sandboxes.Items {
		wwcntr := cw.newSandboxContainer(ctx, sandbox)
		if wwcntr == nil {
			continue
		}
		containers = append(containers, wwcntr)
	}

	return containers, nil
}

// Inspect (only) those container details of interest to us, given the name or
// ID of a container.
func (cw *CRIWatcher) Inspect(ctx context.Context, nameorid string) (*whalewatcher.Container, error) {
	cntrs, err := cw.client.rtcl.ListContainers(ctx, &runtime.ListContainersRequest{
		Filter: &runtime.ContainerFilter{Id: nameorid},
	})
	if err != nil {
		return nil, err
	}
	if len(cntrs.Containers) == 1 {
		return cw.newContainer(ctx, cntrs.Containers[0], nil), nil
	}
	sandboxes, err := cw.client.rtcl.ListPodSandbox(ctx, &runtime.ListPodSandboxRequest{
		Filter: &runtime.PodSandboxFilter{Id: nameorid},
	})
	if err != nil {
		return nil, err
	}
	if len(sandboxes.Items) == 1 {
		return cw.newSandboxContainer(ctx, sandboxes.Items[0]), nil
	}
	return nil, fmt.Errorf("cannot inspect container with id %q", nameorid)
}

// newContainer returns the container details of interest to us. If the
// container is not alive (with a process), then nil is returned instead. If the
// pod sandbox is already known, it can be passed in to avoid an additional CRI
// API roundtrip in order to determine the pod meta data, especially the pod
// name and namespace.
func (cw *CRIWatcher) newContainer(
	ctx context.Context,
	cntr *runtime.Container,
	optPod *runtime.PodSandbox,
) *whalewatcher.Container {
	if cntr.State != runtime.ContainerState_CONTAINER_RUNNING {
		return nil
	}
	// If we didn't get the related pod details, then we need to query them now.
	pods, err := cw.client.rtcl.ListPodSandbox(ctx, &runtime.ListPodSandboxRequest{
		Filter: &runtime.PodSandboxFilter{Id: cntr.PodSandboxId}})
	if err != nil || len(pods.Items) != 1 {
		return nil
	}
	// We still don't know this container's PID and the CRI API actually
	// doesn't provide it anywhere. Instead, at least some CRI-supporting
	// container engines reveal container PIDs through the "info" element of
	// the container status. Well, another round trip to the container
	// engine, then. Thanks CRI for nothing.
	status, err := cw.client.rtcl.ContainerStatus(ctx, &runtime.ContainerStatusRequest{
		ContainerId: cntr.Id,
		Verbose:     true,
	})
	if err != nil {
		return nil
	}
	// Please note that the "info" element inside the verbose information
	// element uses JSON textual representation. This *is* convoluted.
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

	// Shallow clone the labels and ensure that the map isn't nil.
	labels := maps.Clone(cntr.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	// Map annotations to the generic labels, using a unique key prefix to make
	// them easily and deterministically detectable.
	for key, value := range cntr.Annotations {
		labels[AnnotationKeyPrefix+key] = value
	}

	labels[PodUidLabel] = pods.Items[0].Metadata.Uid
	labels[PodNameLabel] = pods.Items[0].Metadata.Name
	labels[PodNamespaceLabel] = pods.Items[0].Metadata.Namespace
	labels[PodContainerNameLabel] = cntr.Metadata.Name

	// If this happens to be a pod sandbox container (in the context of event
	// processing), then mark it as such for convenience.
	if cntr.Id == cntr.PodSandboxId {
		labels[PodSandboxLabel] = "" // exact value doesn't matter
	}

	return &whalewatcher.Container{
		ID:     cntr.Id,
		Name:   cntr.Metadata.Name,
		Labels: labels,
		PID:    innerInfo.PID,
		Paused: false, // there is no pause notion in Kubernetes
	}
}

// newSandboxContainer returns the container details of a pod sandbox of
// interest to us. This is similar to newContainer, but instead is required when
// dealing with the sandbox containers; these are separate from the ordinary
// workload containers in the CRI API architecture.
func (cw *CRIWatcher) newSandboxContainer(
	ctx context.Context,
	sandbox *runtime.PodSandbox,
) *whalewatcher.Container {
	if sandbox.State != runtime.PodSandboxState_SANDBOX_READY {
		return nil
	}
	// We still don't know this sandbox container's PID and the CRI API actually
	// doesn't provide it anywhere. Instead, at least some CRI-supporting
	// container engines reveal container PIDs through the "info" element of the
	// container status. Well, another round trip to the container engine, then.
	// Thanks CRI for nothing.
	status, err := cw.client.rtcl.PodSandboxStatus(ctx, &runtime.PodSandboxStatusRequest{
		PodSandboxId: sandbox.Id,
		Verbose:      true,
	})
	if err != nil {
		return nil
	}
	// Please note that the "info" element inside the verbose information
	// element uses JSON textual representation. This *is* convoluted.
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

	// Shallow clone the labels and ensure that the map isn't nil.
	labels := maps.Clone(sandbox.Labels)
	if labels == nil {
		labels = map[string]string{}
	}
	// Map annotations to the generic labels, using a unique key prefix to make
	// them easily and deterministically detectable.
	for key, value := range sandbox.Annotations {
		labels[AnnotationKeyPrefix+key] = value
	}

	labels[PodUidLabel] = sandbox.Metadata.Uid
	labels[PodNameLabel] = sandbox.Metadata.Name
	labels[PodNamespaceLabel] = sandbox.Metadata.Namespace
	labels[PodContainerNameLabel] = sandbox.Id

	labels[PodSandboxLabel] = "" // exact value doesn't matter

	return &whalewatcher.Container{
		ID:     sandbox.Id,
		Name:   sandbox.Id,
		Labels: labels,
		PID:    innerInfo.PID,
		Paused: false, // there is no pause notion in Kubernetes
	}
}

// LifecycleEvents streams container engine events, limited just to those events
// in the lifecycle of containers getting born (=alive, as opposed to, say,
// “conceived”) and die.
func (cw *CRIWatcher) LifecycleEvents(ctx context.Context) (
	<-chan engineclient.ContainerEvent, <-chan error,
) {
	cntreventstream := make(chan engineclient.ContainerEvent, 16)
	cntrerrstream := make(chan error, 1)

	go func() {
		defer close(cntrerrstream)
		evcl, err := cw.client.rtcl.GetContainerEvents(ctx,
			&runtime.GetEventsRequest{ /*nothing*/ })
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
			// At least in the cases of containerd and cri-o, the sandbox
			// lifcycle also emits container events with their ContainerId equal
			// to the PodSandboxStatus.Id.
			//
			// In case of containerd, please see the code here:
			// https://github.com/containerd/containerd/blob/4d2c8879908285454a4006534cb0af82bb58a406/pkg/cri/server/sandbox_run.go#L506
			switch ev.ContainerEventType {
			case runtime.ContainerEventType_CONTAINER_STARTED_EVENT:
				cntreventstream <- engineclient.ContainerEvent{
					Type: engineclient.ContainerStarted,
					ID:   ev.ContainerId, // use ID to be unambiguous
				}
			case runtime.ContainerEventType_CONTAINER_STOPPED_EVENT:
				cntreventstream <- engineclient.ContainerEvent{
					Type: engineclient.ContainerExited,
					ID:   ev.ContainerId, // use ID to be unambiguous
				}
			}
		}
	}()

	return cntreventstream, cntrerrstream
}
