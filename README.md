# Whalewatcher

[![PkgGoDev](https://pkg.go.dev/badge/github.com/thediveo/whalewatcher)](https://pkg.go.dev/github.com/thediveo/whalewatcher)
[![GitHub](https://img.shields.io/github/license/thediveo/whalewatcher)](https://img.shields.io/github/license/thediveo/whalewatcher)
![build and test](https://github.com/thediveo/whalewatcher/workflows/build%20and%20test/badge.svg?branch=master)
![goroutines](https://img.shields.io/badge/go%20routines-not%20leaking-success)
![file descriptors](https://img.shields.io/badge/file%20descriptors-not%20leaking-success)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/whalewatcher)](https://goreportcard.com/report/github.com/thediveo/whalewatcher)

🔭🐋 `whalewatcher` is a simple Golang module that relieves applications from
the tedious task of constantly monitoring "alive" container workloads: no need
to watching boring event streams or alternatively polling to have the accurate
picture.

Instead, using `whalewatcher` an application simply asks for the current state
of affairs at any time when it needs to do so. The workload state then is
directly answered from `whalewatcher`'s trackers without causing container
engine load: which containers are alive right now? And what composer projects
are in use?

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

> ℹ️ If your application requires immediate action upon container lifecycle
> events then our `whalewatcher` **isn't the right module** for it: this module
> is designed for those use cases where the application needing information
> about containers is completely decoupled from container lifecycle events.

## Features

- tracks container information with respect to a container's ID/name, PID,
  labels, (un)pausing state, and optional (composer) project. See the
  [`whalewatcher.Container`](https://pkg.go.dev/github.com/thediveo/whalewatcher#Container)
  type for details.
- supports different container engines:
  - [Docker/Moby](https://github.com/moby/moby)
  - plain [containerd](https://github.com/containerd/containerd)
  - in the future, when [cri-o/cri-o issue 5609, _[RFE] Event API for container
    lifecycle events_](https://github.com/cri-o/cri-o/issues/5609) hopefully
    will be implemented, [cri-o](https://cri-o.io/) can be supported some day
    too. 
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
mocking of Docker clients to the small extend required for whale watching. The
tests also cover leak checks:

* goroutine leak checking courtesy of Gomega's
  [`gleak`](https://onsi.github.io/gomega/#codegleakcode-finding-leaked-goroutines)
  package.

* file descriptor leak checking courtesy of the
  [@thediveo/fdooze](https://github.com/thediveo/fdooze) module.

> **Note:** do **not run parallel tests** for multiple packages. `make test`
ensures to run all package tests always sequentially, but in case you run `go
test` yourself, please don't forget `-p 1` when testing multiple packages in
one, _erm_, go.

## Copyright and License

Copyright 2021 Harald Albrecht, licensed under the Apache License, Version 2.0.
