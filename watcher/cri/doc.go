/*
Package cri provides a container Watcher for CRI pod event API-supporting
engines.

# Usage

	import "github.com/thediveo/whalewatcher/watcher/cri"
	watcher := cri.NewWatcher("/path/to/CRI-API-endpoint.sock")

Please note that there is no default/standard path for CRI API endpoints;
instead, the exact path depends on the specific container engine and/or the
particular deployment.

The watcher constructor accepts options, with currently the only option being
specifying a container engine's PID. The PID information then can be used
downstream in tools like [lxkns] to translate container PIDs between different
PID namespaces.

[lxkns]: https://github.com/thediveo/lxkns
*/
package cri
