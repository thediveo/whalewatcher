/*
Package ctr provides invoking a “ctr” command inside a Docker container
containing containerd and ctr.

Due to some design decisions on containerd's native API any client that wants to
pull images and then deploy them as containers has to be inside the same kernel
namespaces as the containerd daemon. It is unclear as to which namespace types
need to be the same, but assumed to be the same mount and user namespaces. This
affects both tools such as “cri” as well as the generic API client.

[Successfully] will fail the current test in case “ctr” either cannot be started
or exits with a non-zero result code.

In contrast, [Exec] will only fail the current test in case “ctr” cannot be
started at all, but otherwise will gladly accept all exit code outcomes and
report them back.

Putting these functions into their own public package allows for re-use and
maintaining them in one central place.
*/
package ctr
