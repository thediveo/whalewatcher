// Copyright 2023 Harald Albrecht.
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

package cri

import (
	"github.com/cenkalti/backoff/v4"
	engineclient "github.com/thediveo/whalewatcher/engineclient/cri"
	"github.com/thediveo/whalewatcher/watcher"
)

// Type ID of the container engine handled by this watcher.
const Type = engineclient.Type

// New returns a Watcher for keeping track of the currently alive containers
// (including sandbox containers).
//
// Please note that there is no default value for the CRI API socket path, so it
// must not be the empty string.
//
// If the backoff is nil then the backoff defaults to backoff.StopBackOff, that
// is, any failed operation will never be retried.
//
// Finally, containerd engine client-specific options can be passed in.
func New(criapisock string, buggeroff backoff.BackOff, opts ...engineclient.NewOption) (watcher.Watcher, error) {
	cdclient, err := engineclient.New(criapisock)
	if err != nil {
		return nil, err
	}
	return watcher.New(engineclient.NewCRIWatcher(cdclient, opts...), buggeroff), nil
}
