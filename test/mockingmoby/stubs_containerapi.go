// Copyright 2021 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mockingmoby

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	networktypes "github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// ContainerAttach is not implemented.
func (mm *MockingMoby) ContainerAttach(context.Context, string, container.AttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, errNotImplemented
}

// ContainerCommit is not implemented.
func (mm *MockingMoby) ContainerCommit(context.Context, string, container.CommitOptions) (container.CommitResponse, error) {
	return container.CommitResponse{}, errNotImplemented
}

// ContainerCreate is not implemented.
func (mm *MockingMoby) ContainerCreate(context.Context, *container.Config, *container.HostConfig, *networktypes.NetworkingConfig, *specs.Platform, string) (container.CreateResponse, error) {
	return container.CreateResponse{}, errNotImplemented
}

// ContainerDiff is not implemented.
func (mm *MockingMoby) ContainerDiff(context.Context, string) ([]container.FilesystemChange, error) {
	return nil, errNotImplemented
}

// ContainerExecAttach is not implemented.
func (mm *MockingMoby) ContainerExecAttach(context.Context, string, container.ExecAttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, errNotImplemented
}

// ContainerExecCreate is not implemented.
func (mm *MockingMoby) ContainerExecCreate(context.Context, string, container.ExecOptions) (container.ExecCreateResponse, error) {
	return container.ExecCreateResponse{}, errNotImplemented
}

// ContainerExecInspect is not implemented.
func (mm *MockingMoby) ContainerExecInspect(context.Context, string) (container.ExecInspect, error) {
	return container.ExecInspect{}, errNotImplemented
}

// ContainerExecResize is not implemented.
func (mm *MockingMoby) ContainerExecResize(context.Context, string, container.ResizeOptions) error {
	return errNotImplemented
}

// ContainerExecStart is not implemented.
func (mm *MockingMoby) ContainerExecStart(context.Context, string, container.ExecStartOptions) error {
	return errNotImplemented
}

// ContainerExport is not implemented.
func (mm *MockingMoby) ContainerExport(context.Context, string) (io.ReadCloser, error) {
	return nil, errNotImplemented
}

// ContainerInspectWithRaw is not implemented.
func (mm *MockingMoby) ContainerInspectWithRaw(context.Context, string, bool) (container.InspectResponse, []byte, error) {
	return container.InspectResponse{}, nil, errNotImplemented
}

// ContainerKill is not implemented.
func (mm *MockingMoby) ContainerKill(context.Context, string, string) error {
	return errNotImplemented
}

// ContainerLogs is not implemented.
func (mm *MockingMoby) ContainerLogs(context.Context, string, container.LogsOptions) (io.ReadCloser, error) {
	return nil, errNotImplemented
}

// ContainerPause is not implemented.
func (mm *MockingMoby) ContainerPause(context.Context, string) error {
	return errNotImplemented
}

// ContainerRemove is not implemented.
func (mm *MockingMoby) ContainerRemove(context.Context, string, container.RemoveOptions) error {
	return errNotImplemented
}

// ContainerRename is not implemented.
func (mm *MockingMoby) ContainerRename(context.Context, string, string) error {
	return errNotImplemented
}

// ContainerResize is not implemented.
func (mm *MockingMoby) ContainerResize(context.Context, string, container.ResizeOptions) error {
	return errNotImplemented
}

// ContainerRestart is not implemented.
func (mm *MockingMoby) ContainerRestart(context.Context, string, container.StopOptions) error {
	return errNotImplemented
}

// ContainerStatPath is not implemented.
func (mm *MockingMoby) ContainerStatPath(context.Context, string, string) (container.PathStat, error) {
	return container.PathStat{}, errNotImplemented
}

// ContainerStats is not implemented.
func (mm *MockingMoby) ContainerStats(context.Context, string, bool) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{}, errNotImplemented
}

// ContainerStatsOneShot is not implemented.
func (mm *MockingMoby) ContainerStatsOneShot(context.Context, string) (container.StatsResponseReader, error) {
	return container.StatsResponseReader{}, errNotImplemented
}

// ContainerStart is not implemented.
func (mm *MockingMoby) ContainerStart(context.Context, string, container.StartOptions) error {
	return errNotImplemented
}

// ContainerStop is not implemented.
func (mm *MockingMoby) ContainerStop(context.Context, string, container.StopOptions) error {
	return errNotImplemented
}

// ContainerTop is not implemented.
func (mm *MockingMoby) ContainerTop(context.Context, string, []string) (container.TopResponse, error) {
	return container.TopResponse{}, errNotImplemented
}

// ContainerUnpause is not implemented.
func (mm *MockingMoby) ContainerUnpause(context.Context, string) error {
	return errNotImplemented
}

// ContainerUpdate is not implemented.
func (mm *MockingMoby) ContainerUpdate(context.Context, string, container.UpdateConfig) (container.UpdateResponse, error) {
	return container.UpdateResponse{}, errNotImplemented
}

// ContainerWait is not implemented.
func (mm *MockingMoby) ContainerWait(context.Context, string, container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return nil, nil
}

// CopyFromContainer is not implemented.
func (mm *MockingMoby) CopyFromContainer(context.Context, string, string) (io.ReadCloser, container.PathStat, error) {
	return nil, container.PathStat{}, errNotImplemented
}

// CopyToContainer is not implemented.
func (mm *MockingMoby) CopyToContainer(context.Context, string, string, io.Reader, container.CopyToContainerOptions) error {
	return errNotImplemented
}

// ContainersPrune is not implemented.
func (mm *MockingMoby) ContainersPrune(context.Context, filters.Args) (container.PruneReport, error) {
	return container.PruneReport{}, errNotImplemented
}
