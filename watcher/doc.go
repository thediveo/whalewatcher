/*
Package watcher allows keeping track of the currently alive containers of a
container engine.

Currently, the following container engines are supported:

  - Docker,
  - plain containerd, including nerdctl-project awareness.

# Usage

Creating container watchers for specific container engines preferably should be
done using a particular container engine's NewWatcher convenience function, such
as:

	import "github.com/thediveo/whalewatcher/watcher/moby"
	moby := NewWatcher("")

The engine watcher NewWatcher() constructors additionally accept options. The
only option currently being defines is to specify a container engine's PID. The
PID information then can be used downstream in tools like
github.com/thediveo/lxkns to translate container PIDs between different PID
namespaces. It's up to the API user to supply the correct PIDs where necessary
and known. The watchers themselves do not need the PID information for their own
operations.

# Gory Details Notes

The really difficult part here is to properly synchronize at the beginning with
a container engine's state without getting out of sync: while we get ordered
events (do we actually?!) there's an event horizon (and this ain't Kubernetes)
so we need to run an initial listing of containers. The problem now is that when
events happen while the list is in progress, we don't know how events and
container listing results relate to each other.

To only slightly make matters more complicated, a simple single list request
usually isn't enough, but we need many round trips to a container engine in
order to get our list of containers with the required details (such as labels,
PIDs, and pausing state).

If at any time there is some event happening, then how am we supposed to deal
with the situation? Of course, assuming a lazy container host with but few
events and those events not happening when starting the watcher is one way to
deal with the problem. This is what many tools seem to do – judging from our
code the lazy route doesn't seem so bad after all...

Oh, another complication is that containerd doesn't enforce unique IDs (UIDs) as
Docker does with its ID: in case there is a slow list and a sudden container
death with a rapid resurrection while the list is still going on, then with
containerd and depending on the client creating containers we will see the same
ID reappear. With Docker, we never see the same ID, but maybe only the same
(service) name. It's sad that the clever Docker architecture for UIDs+names did
not carry over to containerd's architecture.

Our watcher thus works as follows: it immediately starts listening to events and
then kicks of listing (and "inspecting") containers. While the listing is going
on, the watcher deals with certain events differently compared to after the
listing has been done and its results processed.

Initially, the watcher remembers all dying container IDs during an ongoing
listing. This dead container list is then used when processing the results of
the full container listing to avoid adding dead containers to the watcher's
final container list.

Similar, container pause state change events are also queued during the time of
an ongoing full container listing. That's because we usually won't know the
details about the state-changing containers yet (unless we were lucky to just
see a container creation event). So we queue the state change events, but
optimize to store only the latest pause state of a container. After the full
container listing is done, we "replay" the queued pause state change events:
this ensures that we end up with the correct pausing state for the containers
that changed their pause states while the listing was in progress.
*/
package watcher
