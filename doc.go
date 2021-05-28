/*

Package whalewatcher watches Docker containers as they come and go from the
perspective of containers that are "alive", that is, only those containers with
actual processes and not "dead" (stopped) containers without any processes.
Additionally, this package understands how such containers optionally are
organized into composer projects (Docker composer,
https://github.com/docker/compose).

As the focus is on containers that are either in running or paused states, the
envisioned use cases are thus tools that somehow interact with processes of
these "alive" containers, especially via various elements of the proc
filesystem.

In order to cause as low load as possible this whalewatcher monitors the
container engine's container lifecycle-related events instead of polling.

Portfolio

The information model starts with the Portfolio: the Portfolio knows about the
currently available projects (ComposerProjects).

ComposerProject

Composer projects are either explicitly named or the "zero" project that has no
name (empty name). Projects then know the Containers associated to them.

Container

Containers store limited aspects about individual containers, such as their
names, IDs, and PIDs.

*/
package whalewatcher
