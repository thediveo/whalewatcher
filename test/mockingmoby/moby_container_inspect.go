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

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// ContainerInspect returns details about a particular mocked container.
func (mm *MockingMoby) ContainerInspect(ctx context.Context, nameorid string, options client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	if err := isCtxCancelled(ctx); err != nil {
		return client.ContainerInspectResult{}, err
	}
	if err := callHook(ctx, ContainerInspectPre); err != nil {
		return client.ContainerInspectResult{}, err
	}
	c, ok := mm.lookup(nameorid)
	if err := callHook(ctx, ContainerInspectPost); err != nil {
		return client.ContainerInspectResult{}, err
	}
	if !ok {
		return client.ContainerInspectResult{}, errwrap(errdefs.ErrNotFound, "no such container %q", nameorid)
	}
	return client.ContainerInspectResult{
		Container: container.InspectResponse{
			ID:   c.ID,
			Name: "/" + c.Name,
			State: &container.State{
				Status:  MockedContainerStates[c.Status],
				Running: c.Status == MockedRunning || c.Status == MockedPaused,
				Paused:  c.Status == MockedPaused,
				Pid:     c.PID,
			},
			Config: &container.Config{
				Labels: c.Labels,
			},
		},
	}, nil
}
