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

	"github.com/docker/docker/api/types"
)

// ContainerList returns the list of currently known containers, ignoring any
// list options.
func (mm *MockingMoby) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	if err := isCtxCancelled(ctx); err != nil {
		return nil, err
	}
	if err := callHook(ctx, ContainerListPre); err != nil {
		return nil, err
	}
	mm.mux.RLock()
	cntrs := make([]types.Container, 0, len(mm.containers))
	for _, c := range mm.containers {
		cntr := types.Container{
			ID:     c.ID,
			Names:  []string{"/" + c.Name},
			Labels: c.Labels,
			State:  MockedStates[c.Status],
			Status: MockedStatus[c.Status],
		}
		cntrs = append(cntrs, cntr)
	}
	mm.mux.RUnlock()
	if err := callHook(ctx, ContainerListPost); err != nil {
		return nil, err
	}
	return cntrs, nil
}
