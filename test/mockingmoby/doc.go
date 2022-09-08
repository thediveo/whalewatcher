/*
Package mockingmoby is a very minimalist Docker mock client designed for simple
unit tests in the whalewatcher package. Only limited listing and inspecting
containers is supported, as well as streaming container start and die events.
Moreover, only very few container properties are mocked, just to the extend
needed in the whalewatcher package.

But in contrast to using a real Docker client in unit tests mockingmoby offers
service API hooks which get called at the beginning and end of service API
calls. This can be used to synchronize certain "asynchronous" events with exact
logical timing without the need to instrument production code under test with
hooks. The service API hooks are passed in via (service) context values.

The mocked containers are not created and destroyed using the standard Docker
client service API but instead using AddContainer and RemoveContainer. In
addition, a mock container can be "stopped" using StopContainer, so it gets into
the "exited" state but still exists.
*/
package mockingmoby
