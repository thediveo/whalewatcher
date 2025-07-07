# Whalewatcher
[![PkgGoDev](https://pkg.go.dev/badge/github.com/thediveo/whalewatcher)](https://pkg.go.dev/github.com/thediveo/whalewatcher)
[![GitHub](https://img.shields.io/github/license/thediveo/whalewatcher)](https://img.shields.io/github/license/thediveo/whalewatcher)
![build and test](https://github.com/thediveo/whalewatcher/workflows/build%20and%20test/badge.svg?branch=master)
[![goroutines](https://img.shields.io/badge/go%20routines-not%20leaking-success)](https://pkg.go.dev/github.com/onsi/gomega/gleak)
[![file descriptors](https://img.shields.io/badge/file%20descriptors-not%20leaking-success)](https://pkg.go.dev/github.com/thediveo/fdooze)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/whalewatcher)](https://goreportcard.com/report/github.com/thediveo/whalewatcher)
![Coverage](https://img.shields.io/badge/Coverage-87.9%25-brightgreen)

🔭🐋 `whalewatcher` is a Go module that relieves applications from the tedious
task of constantly monitoring "alive" container workloads: no need to watching
boring event streams or alternatively polling to have the accurate picture.
Never worry about how you have to properly synchronize to a changing workload at
startup, this is all taken care of for you by `whalewatcher`.

Instead, using `whalewatcher` your application simply asks for the current state
of affairs at any time when it needs to do so. The workload state then is
directly answered from `whalewatcher`'s trackers without causing container
engine load: which containers are alive right now? And what composer projects
are in use?

Alternatively, your application can also consume workload lifecycle events
provided by `whalewatcher`. The benefit of using `whalewatcher` instead of the
plain Docker API is that you get the initial synchronization done properly that
will emit container workload (fake) start events, so you always get the correct
current picture.

Oh, `whalewatcher` isn't limited to just Docker, it also supports other
container engines, namely plain containerd, any CRI+event PLEG supporting
engines (containerd, cri-o), and finally podmand. For podman, read carefully the
notes below.

## Stayin' Alive

This module watches Docker and plain containerd containers becoming "alive" with
processes and later die, keeping track of only the "alive" containers. On
purpose, `whalewatcher` focuses solely on _running_ and _paused_ containers, so
those only that have at least an initial container process running (and thus a
PID).

Thus, use cases for `whalewatcher` are container-aware tools that seemingly
randomly need the current state of affairs for all running containers – such as
[lxkns](https://github.com/thediveo/lxkns). These tools themselves now don't
need anymore to do the ugly lifting of container engine event tracking, engine
state resynchronization after reconnects, et cetera. Here, the `whalewatcher`
module reduces system load especially when state is requested in bursts, as it
offers a load-optimized kind of "cache". Yet this cache is always closely
synchronized to the container engine state.

> ℹ️ This module now optionally supports receiving container lifecycle events by
> requesting a lifecycle event stream from a `watcher.Watcher`. Only the
> lifecycle events are supported for when a container becomes alive or exists,
> or it pauses or unpauses.

## Features

- tracks container information with respect to a container's ID/name, PID,
  labels, (un)pausing state, and optional (composer) project. See the
  [`whalewatcher.Container`](https://pkg.go.dev/github.com/thediveo/whalewatcher#Container)
  type for details.
- two APIs available:
  - query workload situation on demand.
  - workload lifecycle events.
- supports multiple types of container engines:
  - [Docker/Moby](https://github.com/moby/moby).
  - plain [containerd](https://github.com/containerd/containerd) using containerd's native API.
  - [cri-o](https://cri-o.io/) and [containerd](https://github.com/containerd/containerd) via the generic CRI pod event API. In principle, other container engines implementing the CRI pod event API should also work:
    - sandbox container lifecycle events must be reported and not suppressed.
    - sandbox and container PIDs must be reported by the verbose variant of the
      container status API call in the PID field of the JSON info object.
  - Podman: 
    - you will have to **use the Docker/Moby watcher.**
    - Due to several serious unfixed issues we're not supporting Podman's own
      API any longer and have archived the sealwatcher _experiment_. More
      background information can be found in [alias
      podman=p.o.'d.man](http://thediveo.github.io/#/art/podman). To paraphrase
      the podman project's answer: _if you need a stable API, use the Docker
      API_. Got that.
- composer project-aware:
  - [docker-compose](https://docs.docker.com/compose/)
  - [nerdctl](https://github.com/containerd/nerdctl)
- optional configurable automatic retries using
  [backoffs](github.com/cenkalti/backoff) (with different strategies as
  supported by the external backoff module).
- documentation ... please see:
  [![PkgGoDev](https://pkg.go.dev/badge/github.com/thediveo/whalewatcher)](https://pkg.go.dev/github.com/thediveo/whalewatcher)

## Turtlefinder

Depending on your use case, you might want to use
[`@siemens/turtlefinder`](https://github.com/siemens/turtlefinder): it
autodetects the different container engines and then starts the required whale
watchers. The turtlefinder additionally detects container engines inside
containers, and it can also discover and kick the multiple socket-activated
podman daemons for system, users, etc. into life.

## Example Usage

From `example/main.go`: this example starts a watcher for the host's Docker (or
podman) daemon, using the `/run/docker.sock` API endpoint. In this example, we
first wait for the initial synchronization to finish, and afterwards print the
container workload. Please note that only workload with running/paused
containers is shown – that is, the containers with processes.

```go
package main

import (
    "context"
    "fmt"
    "sort"

    "github.com/thediveo/whalewatcher/watcher/moby"
)

func main() {
    // connect to the Docker engine; configure no backoff.
    whalewatcher, err := moby.New("unix:///run/docker.sock", nil)
    if err != nil {
        panic(err)
    }
    ctx, cancel := context.WithCancel(context.Background())
    fmt.Printf("watching engine ID: %s\n", whalewatcher.ID(ctx))

    // run the watch in a separate go routine.
    done := make(chan struct{})
    go func() {
        if err := whalewatcher.Watch(ctx); ctx.Err() != context.Canceled {
            panic(err)
        }
        close(done)
    }()

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
    <-done
    whalewatcher.Close()
}
```

## DevContainer

Do yourself a favor, tinker with this Go module in a devcontainer; this gives
you a controlled and somewhat isolated environment.

> [!CAUTION]
>
> Do **not** use VSCode's "~~Dev Containers: Clone Repository in Container
> Volume~~" command, as it is utterly broken by design, ignoring
> `.devcontainer/devcontainer.json`.

1. `git clone https://github.com/thediveo/irks`
2. in VSCode: Ctrl+Shift+P, "Dev Containers: Open Workspace in Container..."
3. select `irks.code-workspace` and off you go...

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## Copyright and License

`whalewatcher` is Copyright 2021, 2024 Harald Albrecht, licensed under the
Apache License, Version 2.0.
