/*
Package whalewatcher watches Docker and containerd containers as they come and
go from the perspective of containers that are "alive", that is, only those
containers with actual processes. In contrast, freshly created or "dead"
containers without any processes are not tracked.

Furthermore, this package understands how containers optionally are organized
into composer projects [Docker compose]. Please note that full nerdctl
project-awareness currently is blocked by [nerdctl issue #241].

As the focus of this module is on containers that are either in running or
paused states, the envisioned use cases are tools that solely interact with
processes, Linux-kernel namespaces, et cetera of these containers (often via
various elements of the proc filesystem).

In order to cause only as low system load as possible this module monitors the
container engine's container lifecycle-related events instead of stupid polling.
In particular, this module decouples an application's access to the current
state from tracking this container state.

Optionally, applications can subscribe to an events channel that passes on the
lifecycle events whalewatcher receives.

# Watcher

A [github.com/thediveo/whalewatcher/watcher.Watcher] watches the containers of a
single container engine instance when running its Watch method in a separate go
routine. Cancel its passed context to stop watching.

Watchers return information about alive containers (and optionally their
organization into projects) via a Portfolio. Please do not keep the Portfolio
reference for long periods of time, as might change in case the watcher needs to
reconnect to a container engine after losing API contact.

Please refer to example/main.go as an example:

	package main

	import (
	    "context"
	    "fmt"
	    "sort"

	    "github.com/thediveo/whalewatcher/watcher/moby"
	)

	func main() {
		whalewatcher, err := moby.NewWatcher("unix:///var/run/docker.sock")
		if err != nil {
			panic(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		fmt.Printf("watching engine ID: %s\n", whalewatcher.ID(ctx))

		// run the watch in a separate go routine.
		go whalewatcher.Watch(ctx)

		// depending on application you don't need to wait for the first results to
		// become ready; in this example we want to wait for results.
		<-whalewatcher.Ready()

		// get list of projects; we add the unnamed "" project which automatically
		// contains all non-project (standalone) containers.
		projectnames := append(whalewatcher.Portfolio().Names(), "")
		sort.Strings(projectnames)
		for _, projectname := range projectnames {
			containers := whalewatcher.Portfolio().Project(projectname)
			if containers == nil {
				continue // doh ... gone!
			}
			fmt.Printf("project %q:\n", projectname)
			for _, container := range containers.Containers() {
				fmt.Printf("  container %q with PID %d\n", container.Name, container.PID)
			}
			fmt.Println()
		}

		// finally stop the watch
		cancel()
		whalewatcher.Close()
	}

Note: if an application needs to watch both Docker and "pure" containerd
containers, then it needs to create two watchers, one for the Docker engine and
another one for the containerd instance. The containerd watcher doesn't watch
any Docker-managed containers (it cannot as Docker does not attach all
information at the containerd level, especially not the container name).

# Portfolio

The container information model starts with the [Portfolio]: a Portfolio consists
of one or more projects in form of [ComposerProject], including the "unnamed"
ComposerProject (that contains all non-project containers).

# ComposerProject

Composer projects are either explicitly named, or the "zero" project that has no
name (that is, the empty name). A [ComposerProject] contains [Container] objects.

# Container

Containers store limited aspects about individual containers, such as their
names, IDs, and PIDs.

[nerdctl issue #241]: https://github.com/containerd/nerdctl/issues/241
[Docker compose]: https://github.com/docker/compose
*/
package whalewatcher
