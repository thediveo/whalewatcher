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
