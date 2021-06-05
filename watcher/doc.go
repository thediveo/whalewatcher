/*

Package watcher allows keeping track of the currently alive containers of a
container engine.

Currently, the following container engines are supported:

- Docker
- (upcoming: plain containerd with nerdctl-project awareness)

Usage

Creating container watchers for specific container engines preferably should be
done using a particular container engine's NewWatcher convenience function, such
as:

    import "github.com/thediveo/whalewatcher/watcher/moby"
    moby := NewWatcher("")

*/
package watcher
