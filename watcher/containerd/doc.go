/*
Package containerd provides a container Watcher for containerd engines.

# Usage

	import "github.com/thediveo/whalewatcher/watcher/containerd"
	watcher := containerd.NewWatcher("")

The watcher constructor accepts options, with currently the only option being
specifying a container engine's PID. The PID information then can be used
downstream in tools like [lxkns] to translate container PIDs between different
PID namespaces.

[lxkns]: https://github.com/thediveo/lxkns
*/
package containerd
