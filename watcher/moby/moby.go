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

package moby

import (
	"github.com/docker/docker/client"
	mobyengine "github.com/thediveo/whalewatcher/engineclient/moby"
	"github.com/thediveo/whalewatcher/watcher"
)

// Type ID of the container engine handled by this watcher.
const Type = mobyengine.Type

// NewWatcher returns a Watcher for keeping track of the currently alive
// containers, optionally with the composer projects they're associated with.
//
// When the dockersock parameter is left empty then Docker's usual client
// defaults apply, such as trying to pick up the docker host from the
// environment or falling back to the local host's
// "unix:///var/run/docker.sock".
func NewWatcher(dockersock string) (watcher.Watcher, error) {
	clientopts := []client.Opt{
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	}
	if dockersock != "" {
		clientopts = append(clientopts, client.WithHost(dockersock))
	}
	moby, err := client.NewClientWithOpts(clientopts...)
	if err != nil {
		return nil, err
	}
	return watcher.NewWatcher(mobyengine.NewMobyWatcher(moby)), nil
}
