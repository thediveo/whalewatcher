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

package watcher

// pauseState keeps track of the most recent (un)pause state of a container
// identified by its ID.
type pauseState struct {
	ID     string // container ID
	Paused bool   // true, if container is paused.
}

// pendingPauseStates is a list (queue) of pending container (un)pause states,
// keeping only the most recent per particular container ID. Please note that
// this type must not be used from multiple go routines simultaneously, but only
// from a single go routine.
type pendingPauseStates []pauseState

// Add or update the (un)pause state of the container with the given ID.
func (pps *pendingPauseStates) Add(id string, paused bool) {
	for idx, ps := range *pps {
		if ps.ID == id {
			(*pps)[idx].Paused = paused
			return
		}
	}
	*pps = append(*pps, pauseState{ID: id, Paused: paused})
}

// Remove the (un)pause state of the container with the given ID. If there is no
// such ID, then silently ignore the removal attempt.
func (pps *pendingPauseStates) Remove(id string) {
	for idx, ps := range *pps {
		if ps.ID == id {
			last := len(*pps) - 1
			(*pps)[idx] = (*pps)[last] // ...nothing to block from gc here
			*pps = (*pps)[:last]
			return
		}
	}
}
