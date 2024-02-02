# Whalewatcher
[![PkgGoDev](https://pkg.go.dev/badge/github.com/thediveo/whalewatcher)](https://pkg.go.dev/github.com/thediveo/whalewatcher)
[![GitHub](https://img.shields.io/github/license/thediveo/whalewatcher)](https://img.shields.io/github/license/thediveo/whalewatcher)
![build and test](https://github.com/thediveo/whalewatcher/workflows/build%20and%20test/badge.svg?branch=master)
![goroutines](https://img.shields.io/badge/go%20routines-not%20leaking-success)
![file descriptors](https://img.shields.io/badge/file%20descriptors-not%20leaking-success)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/whalewatcher)](https://goreportcard.com/report/github.com/thediveo/whalewatcher)
![Coverage](https://img.shields.io/badge/Coverage-85.7%25-brightgreen)

🔭🐋 `whalewatcher` is a simple Golang module that relieves applications from
the tedious task of constantly monitoring "alive" container workloads: no need
to watching boring event streams or alternatively polling to have the accurate
picture. Never worry about how to properly synchronize to a changing workload at
startup, this is all taken care of for you.

Instead, using `whalewatcher` an application simply asks for the current state
of affairs at any time when it needs to do so. The workload state then is
directly answered from `whalewatcher`'s trackers without causing container
engine load: which containers are alive right now? And what composer projects
are in use?

Oh, `whalewatcher` isn't limited to just Docker, it also supports
[nerdctl](https://github.com/containerd/nerdctl) with _plain_ containerd. Wait,
there's even more: any container engine supporting the [CRI pod event
API](https://github.com/kubernetes/cri-api/blob/604407e718bd257069ddd48932eefea00eb1c9a7/pkg/apis/runtime/v1/api.proto#L123)
can be tracked, too.

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
- supports multiple types of container engines:
  - [Docker/Moby](https://github.com/moby/moby).
  - plain [containerd](https://github.com/containerd/containerd) using containerd's native API.
  - [cri-o](https://cri-o.io/) and [containerd](https://github.com/containerd/containerd) via the generic CRI pod event API. In principle, other container engines implementing the CRI pod event API should also work:
    - sandbox container lifecycle events must be reported and not suppressed.
    - sandbox and container PIDs must be reported by the verbose variant of the
      container status API call in the PID field of the JSON info object.
  - Podman: **use the Docker/Moby watcher.** Due to several serious unfixed
    issues we're not supporting Podman's own API any longer and have archived
    the sealwatcher _experiment_. More background information can be found in
    [alias podman=p.o.'d.man](http://thediveo.github.io/#/art/podman). To
    paraphrase the podman project's answer: _if you need a stable API, use the
    Docker API_. Got that.
- composer project-aware:
  - [docker-compose](https://docs.docker.com/compose/)
  - [nerdctl](https://github.com/containerd/nerdctl)
- optional configurable automatic retries using
  [backoffs](github.com/cenkalti/backoff) (with different strategies as
  supported by the external backoff module).
- documentation ... please see:
  [![PkgGoDev](https://pkg.go.dev/badge/github.com/thediveo/whalewatcher)](https://pkg.go.dev/github.com/thediveo/whalewatcher)

## Example Usage

From `example/main.go`:

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
    whalewatcher, err := moby.New("unix:///var/run/docker.sock", nil)
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

## Hacking It

This project comes with comprehensive unit tests, including (albeit limited)
mocking of Docker clients to the small extend required for whale watching.

- unit tests require Docker CE in a _moderately_ recent version. Debian users
  are advised to install Docker CE from Docker's package, as Debian's own
  packages tend to completely outdate function-wise over the lifespan of a
  particular Debian release.

> **Fun Fact:** the tests covering containerd and CRI-O use a dockerized
> container/cri-o image, leveraging the `kindest/base` image by the KinD SIG, in
> ways the SIG surely didn't envision.

The tests come with integrated leak checks:

* goroutine leak checking courtesy of Gomega's
  [`gleak`](https://onsi.github.io/gomega/#codegleakcode-finding-leaked-goroutines)
  package.

* file descriptor leak checking courtesy of the
  [@thediveo/fdooze](https://github.com/thediveo/fdooze) module.

> **Note:** do **not run parallel tests** for multiple packages. `make test`
ensures to run all package tests always sequentially, but in case you run `go
test` yourself, please don't forget `-p 1` when testing multiple packages in
one, _erm_, go.

Unit tests about interfacing with and tracking containerd and CRI container
engines use **Docker containers with containerized containerd and cri-o
engines**. The corresponding test images base on [`kindest/base` Docker
images](https://hub.docker.com/r/kindest/base), courtesy of the [KinD k8s
SIG](https://github.com/kubernetes-sigs/kind). Now, we fully understand that
we're on our own here with no guarantees given by the KinD k8s SIG. However,
their `kindest/base` images are really helpful in coming up with containerizing
a containerd engine to a Docker container that we cannot simply pass by them.

All we're adding is some slim configuration so that we can create some pods
and/or containers. The cri-o bases on some instructions about how to modify the
KinD images to use cri-o instead of containerd; but in our case we install them
both side-by-side. And as it happens, they seem to somehow get along with each
other when confined to the same Docker container.

## VSCode Tasks

The included `go-plugger.code-workspace` defines the following tasks:

- **View Go module documentation** task: installs `pkgsite`, if not done already
  so, then starts `pkgsite` and opens VSCode's integrated ("simple") browser to
  show the go-plugger/v2 documentation.

- **Build workspace** task: builds all, including the shared library test
  plugin.

- **Run all tests with coverage** task: does what it says on the tin and runs
  all tests with coverage.

#### Aux Tasks

- _pksite service_: auxilliary task to run `pkgsite` as a background service
  using `scripts/pkgsite.sh`. The script leverages browser-sync and nodemon to
  hot reload the Go module documentation on changes; many thanks to @mdaverde's
  [_Build your Golang package docs
  locally_](https://mdaverde.com/posts/golang-local-docs) for paving the way.
  `scripts/pkgsite.sh` adds automatic installation of `pkgsite`, as well as the
  `browser-sync` and `nodemon` npm packages for the local user.
- _view pkgsite_: auxilliary task to open the VSCode-integrated "simple" browser
  and pass it the local URL to open in order to show the module documentation
  rendered by `pkgsite`. This requires a detour via a task input with ID
  "_pkgsite_".

## Make Targets

- `make`: lists all targets.
- `make coverage`: runs all tests with coverage and then **updates the coverage
  badge in `README.md`**.
- `make pkgsite`: installs [`x/pkgsite`](golang.org/x/pkgsite/cmd/pkgsite), as
  well as the [`browser-sync`](https://www.npmjs.com/package/browser-sync) and
  [`nodemon`](https://www.npmjs.com/package/nodemon) npm packages first, if not
  already done so. Then runs the `pkgsite` and hot reloads it whenever the
  documentation changes.
- `make report`: installs
  [`@gojp/goreportcard`](https://github.com/gojp/goreportcard) if not yet done
  so and then runs it on the code base.
- `make test`: runs **all** tests (including dynamic plugins).

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md).

## Copyright and License

`whalewatcher` is Copyright 2021, 2024 Harald Albrecht, licensed under the
Apache License, Version 2.0.
