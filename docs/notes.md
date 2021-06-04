# Technical Notes

## Docker Container Properties

### When Listing

The container "description"
[types.Container](https://pkg.go.dev/github.com/docker/docker/api/types#Container)
returned by
[Client.ContainerList](https://pkg.go.dev/github.com/ph/moby/client#Client.ContainerList)
when listing containers.

- **ID**: unique container identifier (hash).
- **Names**: there's always at least one name assigned by the Docker engine; either
  due to explicit naming by an API user or implicitly by the Docker engine when
  only an empty name was specified.
- **Created**: Unix epoch timestamp in seconds (_only_). Please note that
  inspection of a container reveals a more precise creation timestamp.
- **State**: the container state in form of a string with well-known
  (enumeration) values:
  - `running` – of interest to us here, as this container will have a PID of its
    initial container process. (But the PID must be "inspected" separately.)
  - `paused` – of interest to us here, as this container will have a PID of its
    initial container process. (But the PID must be "inspected" separately.)
  - `exited`
  - `dead`
  - `created`
  - `restarting`
  - `removing`
- _Status_: a user-oriented textual status, such as "`Up 42 seconds`" – not of
  much use to us in our context.

> ⚠️ When listing, the returned information lacks certain pieces such as the PID
> of a container's initial process.

### When Inspecting

The container details
[types.ContainerJSON](https://pkg.go.dev/github.com/docker/docker/api/types#ContainerJSON)
(with
[types.ContainerJSONBase](https://pkg.go.dev/github.com/docker/docker/api/types#ContainerJSONBase))
returned by
[Client.ContainerInspect](https://pkg.go.dev/github.com/ph/moby/client#Client.ContainerInspect)
when inspecting containers.

- **ID**
- **Name**: ⚠️ as opposed to _Names_ when listing container "descriptions".
- _Created_: ⚠️ stringified timestamp in ISO UTC format with sub-second
    precision.
- **State**:
  - **Status** ⚠️ corresponds with _State_ (sic!) when listing containers.
  - **Running**: true if running or paused.
  - **Paused**: true if paused (while running).
  - **StartedAt**: stringified timestamp in ISO UTC format with sub-second
    precision.
  - _FinishedAt_: stringified timestamp in ISO UTC format with sub-second
    precision.
- **Pid**: non-zero if container has initial process "_and more..._"
- **Config**
  ([container.Config](https://pkg.go.dev/github.com/docker/docker@v20.10.6+incompatible/api/types/container#Config)):
  - **Labels**: labels assigned to container.

## Docker Events

Events streaming from
[Client.Events](https://pkg.go.dev/github.com/ph/moby/client#Client.Events) when
watching whales. See also:
[events.Message](https://pkg.go.dev/github.com/docker/docker/api/types/events#Message).

### Event Timestamp

- **Time**: Unix epoch timestamp in seconds.
- **TimeNano**: nanoseconds part of event timestamp.

### Container Events

- `id` and `Actor.ID`: container ID (not: name).
- `Type`: "`container`"
- `Action`:
  - **start**: initial container process started.
    - `Attributes`:
      - `name`: container name.
      - _foo_: additional user-specified label name _foo_ and value.
  - **die**: initial container process terminated.
    - `Attributes`:
      - `name`: container name.
      - `exitCode`: initial container process exit code.
      - _foo_: additional user-specified label name _foo_ and value.
  - pause
  - unpause
  - kill
  - stop
  - create
  - destroy
  - ...
