# Whalewatcher

[![PkgGoDev](https://pkg.go.dev/badge/github.com/thediveo/whalewatcher)](https://pkg.go.dev/github.com/thediveo/whalewatcher)
[![GitHub](https://img.shields.io/github/license/thediveo/whalewatcher)](https://img.shields.io/github/license/thediveo/whalewatcher)
![build and test](https://github.com/thediveo/whalewatcher/workflows/build%20and%20test/badge.svg?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/whalewatcher)](https://goreportcard.com/report/github.com/thediveo/whalewatcher)

`whalewatcher` is a simple Golang module that watches Docker and plain
containerd containers becoming alive with processes and later die, keeping track
of only the "alive" containers. On purpose, this module focuses solely on
_running_ and _paused_ containers, so those only that have at least an initial
container process running (and thus a PID).

Envisioned use cases are container-aware tools – such as
[lxkns](https://github.com/thediveo/lxkns) – that seemingly randomly need the
current state of running container affairs. Tools that yet do not want to do the
ugly lifting of container engine event tracking, engine state resynchronization
after reconnects, et cetera. Here, the `whalewatcher` module reduces system load
especially when state is requested in bursts, as it offers a load-optimized kind
of "cache". Yet this cache is always closely synchronized to the container
engine state.

> 🛈 If your application requires immediate action upon container lifecycle
> events then our `whalewatcher` isn't the right module for it.

## Features

- supports different container engines:
  - [Docker/Moby](https://github.com/moby/moby)
  - plain [containerd](https://github.com/containerd/containerd)
- composer project-aware:
  - [docker-compose](https://docs.docker.com/compose/)
  - `nerdctl` composer-project support currently is blocked due to [nerdctl
    issue &#35;241](https://github.com/containerd/nerdctl/issues/241), so
    project-related containers appear as standalone containers.
- documentation ...
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
```

## Hacking It

This project comes with comprehensive unit tests, including (limited) mocking of
Docker clients to the small extend required for whale watching.

## Copyright and License

Copyright 2021 Harald Albrecht, licensed under the Apache License, Version 2.0.
