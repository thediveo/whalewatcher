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

import "context"

// HookKey identifies a particular API pre or post hook.
type HookKey string

const (
	ContainerListPre     = HookKey("containerlistpre")
	ContainerListPost    = HookKey("containerlistpost")
	ContainerInspectPre  = HookKey("containerinspectpre")
	ContainerInspectPost = HookKey("containerinspectpost")
)

// Hook is a hook function called in the processing of a service API request.
// Hook functions get passed HookKeys for the specific types of API requests.
// Hooks might return errors in order to either early abort an API request or
// make it override the API error return value (where applicable).
//
// Please note: hooks never get called for API requests that were already called
// with a "Done" context (cancelled or timed out).
type Hook func(HookKey) error

// WithHook returns a new context with the specific Hook added.
func WithHook(ctx context.Context, key HookKey, hook Hook) context.Context {
	return context.WithValue(ctx, key, hook)
}

// callHook calls the specific type of Hook, if currently registered in this
// context.
func callHook(ctx context.Context, key HookKey) error {
	if h := ctx.Value(key); h != nil {
		return h.(Hook)(key)
	}
	return nil
}
