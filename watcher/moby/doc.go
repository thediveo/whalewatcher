/*

Package moby provides a container Watcher for Docker/Moby engines.

Usage

    import "github.com/thediveo/whalewatcher/watcher/moby"
    watcher := moby.NewWatcher("")

The watcher constructor accepts options, with currently the only option being
specifying a container engine's PID. The PID information then can be used
downstream in tools like github.com/thediveo/lxkns to translate container PIDs
between different PID namespaces.

*/
package moby
