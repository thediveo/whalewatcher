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
	"github.com/docker/docker/api/types/registry"
)

// RegistryLogin is not implemented.
func (mm *MockingMoby) RegistryLogin(ctx context.Context, auth registry.AuthConfig) (registry.AuthenticateOKBody, error) {
	return registry.AuthenticateOKBody{}, errNotImplemented
}

// DiskUsage is not implemented.
func (mm *MockingMoby) DiskUsage(ctx context.Context, options types.DiskUsageOptions) (types.DiskUsage, error) {
	return types.DiskUsage{}, errNotImplemented
}

// Ping is not implemented.
func (mm *MockingMoby) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, errNotImplemented
}
