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

	"github.com/moby/moby/api/types/system"
	"github.com/moby/moby/client"
)

// Info returns engine information, consisting only of a fake engine ID, but
// nothing else.
func (mm *MockingMoby) Info(ctx context.Context, options client.InfoOptions) (client.SystemInfoResult, error) {
	if err := isCtxCancelled(ctx); err != nil {
		return client.SystemInfoResult{}, err
	}
	return client.SystemInfoResult{
		Info: system.Info{
			ID:            "MOCK:MOBY:MOCK:MOBY:MOCK:MOBY:MOCK:MOBY:MOCK:MOBY:MOCK:MOBY",
			ServerVersion: "42.66.6",
		},
	}, nil
}
