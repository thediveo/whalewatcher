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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thediveo/whalewatcher/engineclient"
)

var _ = Describe("BeAContainerEvent matcher", func() {

	It("matches", func() {
		cev := engineclient.ContainerEvent{
			Type:    engineclient.ContainerStarted,
			ID:      "ID42",
			Project: "P",
		}
		Expect(cev).To(BeAContainerEvent(HaveID("ID42"), HaveEventType(engineclient.ContainerStarted)))
		Expect(cev).NotTo(BeAContainerEvent(HaveID("ID42"), HaveEventType(engineclient.ContainerExited)))
	})

	It("properly fails for an unexpected type of actual", func() {
		Expect(BeAContainerEvent(HaveID("ID42")).Match("foo")).Error().To(HaveOccurred())
	})

})
