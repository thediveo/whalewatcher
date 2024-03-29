/*
Package cri implements the CRI API EngineClient. It requires a CRI engine
supporting the GetContainerEvents API in the CRI RuntimeService [Evented PLEG].
When used with the matching watcher, clients can track the container(!) workload
in Kubernetes configurations with low overhead in the steady state.

The CRI EngineClient is not meant as a replacement for the k8s control plane
API. In fact, it can't, as the design of the CRI API doesn't allow so.

This package is now the source of truth for the definitions of label keys used
to report pod sandbox and container meta data, retiring the lxkns kuhbernetes
decorator definitions.

# CRI Notes

The CRI API is primarily designed to make Kubernetes (and its “kubelets” in
particular) happy, not 3rd party tools. This is especially true when it comes to
pod lifecycle events: “The overarching goal of this effort is to reduce the
Kubelet and CRI implementation's steady state CPU usage” ([3386-KEP]).

The unfortunate effect is that we either have to live with what the CRI API has
on offer or resort to engine-specific individual APIs. In case of the CRI-O
engine, there really isn't a specific API except the CRI API.

Also, while Docker and containerd function very well as their own bosses and
have well-equipped APIs to go with that, the CRI API has the division between
the Kubernetes control plane and the container engine baked into it.

For instance, while the CRI API allows attaching labels and annotations to
(sandbox) pods and containers, the real place to store k8s resource(!) labels
and annotations is still Kubernetes' control plane. For some reason, the
pod/container lifecycle event API does not send events when the labels and/or
annotations get changed; that might be due to the assumption that the kubelet
triggered the changes, so it doesn't need to know ... but then, why is this
information evented in other situations? Unfortunately, members of the Evented
PLEG weren't so far clarifying this situation when asked about it.

# Tested CRI API Supporters

The following container engines are covered by the unit tests in this package:
  - [containerd]
  - [cri-o] (please see section below)

# CRI-O

Ensure to enable “pod events” in the container engine configuration:

	# /etc/crio/crio.toml
	[crio.runtime]
	enable_pod_events=true

# Kubernetes Labels and Annotations

Kubernetes differentiates between non-identifying [annotations] and often
identifying [labels] that can be attached to all kinds of resources, and not
just containers. However, the whalewatcher model doesn't differentiate between
annotations and labels as separate first-class elements, but instead maps
annotations also to labels.

In order to avoid potential key clashes of annotations with other labels, we
simply prefix all annotation keys with “annotation.k8s/”. Yes, that's “k8s” and
not “k8s.io”, as Kubernetes deserves its own TLD anyway and we don't want to
mess with the “k8s.io” domain.

# CRI API Model

The [CRI API] obviously has been designed to primarily serve the needs of
(crying) kubelets. Originally a purely REST-type API, the addition of container
event streams is more recent. Wiring up the CRI API to a CRI-type whalewatcher
is more cumbersome than compared with Docker and containerd – and even podman's
own API, which tells a lot.

The so-called “runtime“ service API mostly revolves around these two first-class
runtime elements:
  - pod sandboxes
  - containers

From the perspective of a CRI-type whalewatcher we need the following tidbits of
information:
  - container ID
  - container name
  - containing pod namespace and name (that is, “namespace/name”)
  - container PID (basically the ealdorman PID) – this actually is a very weak
    spot in CRI.
  - container state – which will always be “running” as there is no “pause”
    notion in Kubernetes/CRI.

# Whalewatchers

The arche-typical implementation of whalewatchers needs to not only handle the
flow of container lifecycle events, but also initially get the full picture
about the living containers. And while it gets the full picture (which usually
won't be atomic), events might already change the yet incomplete picture.

The dualism of “getting the full picture” and “lifecycle events” thus means that
we need to deal with several CRI API services – and these different services
like to make our lives a proper misery by ensuring to always return only an
always different subset of the information we actually need.

# Listing

The CRI ListContainers service gives us the container IDs and names, as well as
their labels and annotation, filtered for only running containers. We thus lack
details about the containing pods (only the pod IDs) and the container PIDs are
also missing.

The pod names and namespaces must thus be retrieved separately using the CRI
ListPodSandbox service. It can filter to a specific sandbox ID and then works
like a pod sandbox “inspect”. Unfortunately, we're still short of the PIDs.

As it turns out, container PIDs aren't something the kubelet is interested in
and in unfortunate consequence CRI API providers (that is, container engines)
aren't required to provide such information. On the positive side, the following
CRI-supporting container engines are known to currently provide PID information:
  - containerd
  - CRI-O, see [CRI-O issue #1752]

To get the PID, the CRI ContainerStatus service must be used; it takes a
specific container ID and its “verbose” flag must be true. Otherwise, the result
“info” map won't get populated. The PID is then inside the “info” dictionary
inside the “info” map. Yes, for whatever reason, this is turtles all the way
down.

To sum up:
  - ListContainers (all running, that is)
  - ListPodSandbox (per running container)
  - ContainerStatus (also per running container)

Atari just called and wants its Pong back.

# Lifecycle Events

CRI's GetContainerEvents throws lots of details our way. At this time, there is
no filtering in the publisher provided. For our purposes, we're interested only
in the following to container event types:
  - CONTAINER_STARTED_EVENT
  - CONTAINER_STOPPED_EVENT

But then, we get details we're highly interested in, because events carry both
container status and sandbox status:
  - pod name and namespace
  - container ID
  - container name

But we're still short of the container PID, so we need to get these through an
extra ContainerStatus API call.

[labels]: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
[CRI API]: https://github.com/kubernetes/cri-api/blob/master/pkg/apis/runtime/v1/api.proto
[CRI-O issue #1752]: https://github.com/cri-o/cri-o/issues/1752
[annotations]: https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/
[containerd]: https://containerd.io
[cri-o]: https://cri-o.io
[3386-KEP]: https://github.com/kubernetes/enhancements/blob/master/keps/sig-node/3386-kubelet-evented-pleg/README.md
[Evented PLEG]: https://kubernetes.io/docs/tasks/administer-cluster/switch-to-evented-pleg/
*/
package cri
