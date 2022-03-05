// Copyright 2022 Harald Albrecht.
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

package matcher

import (
	o "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/thediveo/whalewatcher/engineclient"
)

// BeAContainerEvent succeeds when the actual value is a ContainerEvent and
// additionally all passed matchers also succeed.
func BeAContainerEvent(matchers ...types.GomegaMatcher) types.GomegaMatcher {
	return o.WithTransform(func(actual engineclient.ContainerEvent) engineclient.ContainerEvent {
		return actual // Gomega already did the type checking for us ;)
	}, o.SatisfyAll(matchers...))
}

// HaveEventType succeeds if the actual value has a "Type" field with the
// specified ContainerEventType value.
func HaveEventType(evtype engineclient.ContainerEventType) types.GomegaMatcher {
	return o.HaveField("Type", evtype)
}
