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

	"github.com/moby/moby/client"
)

// ContainerAttach is not implemented.
func (mm *MockingMoby) ContainerAttach(context.Context, string, client.ContainerAttachOptions) (client.ContainerAttachResult, error) {
	return client.ContainerAttachResult{}, errNotImplemented
}

// ContainerCommit is not implemented.
func (mm *MockingMoby) ContainerCommit(context.Context, string, client.ContainerCommitOptions) (client.ContainerCommitResult, error) {
	return client.ContainerCommitResult{}, errNotImplemented
}

// ContainerCreate is not implemented.
func (mm *MockingMoby) ContainerCreate(context.Context, client.ContainerCreateOptions) (client.ContainerCreateResult, error) {
	return client.ContainerCreateResult{}, errNotImplemented
}

// ContainerDiff is not implemented.
func (mm *MockingMoby) ContainerDiff(context.Context, string, client.ContainerDiffOptions) (client.ContainerDiffResult, error) {
	return client.ContainerDiffResult{}, errNotImplemented
}

// ContainerExport is not implemented.
func (mm *MockingMoby) ContainerExport(context.Context, string, client.ContainerExportOptions) (client.ContainerExportResult, error) {
	return nil, errNotImplemented
}

// ContainerKill is not implemented.
func (mm *MockingMoby) ContainerKill(context.Context, string, client.ContainerKillOptions) (client.ContainerKillResult, error) {
	return client.ContainerKillResult{}, errNotImplemented
}

// ContainerLogs is not implemented.
func (mm *MockingMoby) ContainerLogs(context.Context, string, client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
	return nil, errNotImplemented
}

// ContainerPause is not implemented.
func (mm *MockingMoby) ContainerPause(context.Context, string, client.ContainerPauseOptions) (client.ContainerPauseResult, error) {
	return client.ContainerPauseResult{}, errNotImplemented
}

// ContainerRemove is not implemented.
func (mm *MockingMoby) ContainerRemove(context.Context, string, client.ContainerRemoveOptions) (client.ContainerRemoveResult, error) {
	return client.ContainerRemoveResult{}, errNotImplemented
}

// ContainerRename is not implemented.
func (mm *MockingMoby) ContainerRename(context.Context, string, client.ContainerRenameOptions) (client.ContainerRenameResult, error) {
	return client.ContainerRenameResult{}, errNotImplemented
}

// ContainerResize is not implemented.
func (mm *MockingMoby) ContainerResize(context.Context, string, client.ContainerResizeOptions) (client.ContainerResizeResult, error) {
	return client.ContainerResizeResult{}, errNotImplemented
}

// ContainerRestart is not implemented.
func (mm *MockingMoby) ContainerRestart(context.Context, string, client.ContainerRestartOptions) (client.ContainerRestartResult, error) {
	return client.ContainerRestartResult{}, errNotImplemented
}

// ContainerStatPath is not implemented.
func (mm *MockingMoby) ContainerStatPath(context.Context, string, client.ContainerStatPathOptions) (client.ContainerStatPathResult, error) {
	return client.ContainerStatPathResult{}, errNotImplemented
}

// ContainerStats is not implemented.
func (mm *MockingMoby) ContainerStats(context.Context, string, client.ContainerStatsOptions) (client.ContainerStatsResult, error) {
	return client.ContainerStatsResult{}, errNotImplemented
}

// ContainerStart is not implemented.
func (mm *MockingMoby) ContainerStart(context.Context, string, client.ContainerStartOptions) (client.ContainerStartResult, error) {
	return client.ContainerStartResult{}, errNotImplemented
}

// ContainerStop is not implemented.
func (mm *MockingMoby) ContainerStop(context.Context, string, client.ContainerStopOptions) (client.ContainerStopResult, error) {
	return client.ContainerStopResult{}, errNotImplemented
}

// ContainerTop is not implemented.
func (mm *MockingMoby) ContainerTop(context.Context, string, client.ContainerTopOptions) (client.ContainerTopResult, error) {
	return client.ContainerTopResult{}, errNotImplemented
}

// ContainerUnpause is not implemented.
func (mm *MockingMoby) ContainerUnpause(context.Context, string, client.ContainerUnpauseOptions) (client.ContainerUnpauseResult, error) {
	return client.ContainerUnpauseResult{}, errNotImplemented
}

// ContainerUpdate is not implemented.
func (mm *MockingMoby) ContainerUpdate(context.Context, string, client.ContainerUpdateOptions) (client.ContainerUpdateResult, error) {
	return client.ContainerUpdateResult{}, errNotImplemented
}

// ContainerWait is not implemented.
func (mm *MockingMoby) ContainerWait(context.Context, string, client.ContainerWaitOptions) client.ContainerWaitResult {
	return client.ContainerWaitResult{}
}

// CopyFromContainer is not implemented.
func (mm *MockingMoby) CopyFromContainer(context.Context, string, client.CopyFromContainerOptions) (client.CopyFromContainerResult, error) {
	return client.CopyFromContainerResult{}, errNotImplemented
}

// CopyToContainer is not implemented.
func (mm *MockingMoby) CopyToContainer(context.Context, string, client.CopyToContainerOptions) (client.CopyToContainerResult, error) {
	return client.CopyToContainerResult{}, errNotImplemented
}

// ContainersPrune is not implemented.
func (mm *MockingMoby) ContainersPrune(context.Context, client.ContainerPruneOptions) (client.ContainerPruneResult, error) {
	return client.ContainerPruneResult{}, errNotImplemented
}
