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
	"time"

	"github.com/docker/docker/api/types"
	containertypes "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	networktypes "github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

func (mm *MockingMoby) ContainerAttach(ctx context.Context, container string, options types.ContainerAttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, errNotImplemented
}

func (mm *MockingMoby) ContainerCommit(ctx context.Context, container string, options types.ContainerCommitOptions) (types.IDResponse, error) {
	return types.IDResponse{}, errNotImplemented
}

func (mm *MockingMoby) ContainerCreate(ctx context.Context, config *containertypes.Config, hostConfig *containertypes.HostConfig, networkingConfig *networktypes.NetworkingConfig, platform *specs.Platform, containerName string) (containertypes.ContainerCreateCreatedBody, error) {
	return containertypes.ContainerCreateCreatedBody{}, errNotImplemented
}

func (mm *MockingMoby) ContainerDiff(ctx context.Context, container string) ([]containertypes.ContainerChangeResponseItem, error) {
	return nil, errNotImplemented
}

func (mm *MockingMoby) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, errNotImplemented
}

func (mm *MockingMoby) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
	return types.IDResponse{}, errNotImplemented
}

func (mm *MockingMoby) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	return types.ContainerExecInspect{}, errNotImplemented
}

func (mm *MockingMoby) ContainerExecResize(ctx context.Context, execID string, options types.ResizeOptions) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerExecStart(ctx context.Context, execID string, config types.ExecStartCheck) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerExport(ctx context.Context, container string) (io.ReadCloser, error) {
	return nil, errNotImplemented
}

func (mm *MockingMoby) ContainerInspectWithRaw(ctx context.Context, container string, getSize bool) (types.ContainerJSON, []byte, error) {
	return types.ContainerJSON{}, nil, errNotImplemented
}

func (mm *MockingMoby) ContainerKill(ctx context.Context, container, signal string) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerLogs(ctx context.Context, container string, options types.ContainerLogsOptions) (io.ReadCloser, error) {
	return nil, errNotImplemented
}

func (mm *MockingMoby) ContainerPause(ctx context.Context, container string) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerRemove(ctx context.Context, container string, options types.ContainerRemoveOptions) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerRename(ctx context.Context, container, newContainerName string) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerResize(ctx context.Context, container string, options types.ResizeOptions) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerRestart(ctx context.Context, container string, timeout *time.Duration) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerStatPath(ctx context.Context, container, path string) (types.ContainerPathStat, error) {
	return types.ContainerPathStat{}, errNotImplemented
}

func (mm *MockingMoby) ContainerStats(ctx context.Context, container string, stream bool) (types.ContainerStats, error) {
	return types.ContainerStats{}, errNotImplemented
}

func (mm *MockingMoby) ContainerStatsOneShot(ctx context.Context, container string) (types.ContainerStats, error) {
	return types.ContainerStats{}, errNotImplemented
}

func (mm *MockingMoby) ContainerStart(ctx context.Context, container string, options types.ContainerStartOptions) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerStop(ctx context.Context, container string, timeout *time.Duration) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerTop(ctx context.Context, container string, arguments []string) (containertypes.ContainerTopOKBody, error) {
	return containertypes.ContainerTopOKBody{}, errNotImplemented
}

func (mm *MockingMoby) ContainerUnpause(ctx context.Context, container string) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainerUpdate(ctx context.Context, container string, updateConfig containertypes.UpdateConfig) (containertypes.ContainerUpdateOKBody, error) {
	return containertypes.ContainerUpdateOKBody{}, errNotImplemented
}

func (mm *MockingMoby) ContainerWait(ctx context.Context, container string, condition containertypes.WaitCondition) (<-chan containertypes.ContainerWaitOKBody, <-chan error) {
	return nil, nil
}

func (mm *MockingMoby) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, types.ContainerPathStat, error) {
	return nil, types.ContainerPathStat{}, errNotImplemented
}

func (mm *MockingMoby) CopyToContainer(ctx context.Context, container, path string, content io.Reader, options types.CopyToContainerOptions) error {
	return errNotImplemented
}

func (mm *MockingMoby) ContainersPrune(ctx context.Context, pruneFilters filters.Args) (types.ContainersPruneReport, error) {
	return types.ContainersPruneReport{}, errNotImplemented
}
