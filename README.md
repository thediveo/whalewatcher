# Whalewatcher

A simple Golang module to watch as Docker containers become alive (with
container processes) and later die, keeping track of who's who. The focus is
solely on containers that are _alive_ (running or paused) so there's at least an
initial container process PID to play around with.

The whalewatcher module is aware of [Docker Compose
Projects](https://docs.docker.com/compose/) and groups containers marked as
services of composer projects into project groups.

This project comes with comprehensive unit tests, including (limited) mocking of
Docker clients to the small extend required for whale watching.

## Copyright and License

Copyright 2021 Harald Albrecht, licensed under the Apache License, Version 2.0.
