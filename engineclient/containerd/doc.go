/*
Package containerd implements the containerd EngineClient.

# Notes

containerd's task states of pausing and paused are both mapped to a paused
container from the perspective of the whalewatcher module.

This engine client by default ignores the "moby" and "k8s.io" namespaces.
*/
package containerd
