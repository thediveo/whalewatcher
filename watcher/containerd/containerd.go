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

package containerd

import (
	"github.com/cenkalti/backoff/v4"
	"github.com/containerd/containerd"
	cdengine "github.com/thediveo/whalewatcher/engineclient/containerd"
	"github.com/thediveo/whalewatcher/watcher"
)

// Type ID of the container engine handled by this watcher.
const Type = cdengine.Type

// New returns a Watcher for keeping track of the currently alive
// containers, optionally with the (nerdctl) composer projects they're
// associated with.
//
// When the containerdsock parameter is left empty then containerd's default
// "/run/containerd/containerd.sock" applies.
//
// If the backoff is nil then the backoff defaults to backoff.StopBackOff, that
// is, any failed operation will never be retried.
//
// Finally, containerd engine client-specific options can be passed in.
func New(containerdsock string, buggeroff backoff.BackOff, opts ...cdengine.NewOption) (watcher.Watcher, error) {
	if containerdsock == "" {
		containerdsock = "/run/containerd/containerd.sock"
	}
	cdclient, err := containerd.New(containerdsock)
	if err != nil {
		return nil, err
	}
	return watcher.New(cdengine.NewContainerdWatcher(cdclient, opts...), buggeroff), nil
}
