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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	mockingMoby = MockedContainer{
		ID:     "1234567890",
		Name:   "mocking_moby",
		Status: MockedCreated,
		PID:    0,
		Labels: map[string]string{"motto": "I'm not dead yet"},
	}

	furiousFuruncle = MockedContainer{
		ID:     "6666666666",
		Name:   "furious_furuncle",
		Status: MockedRunning,
		PID:    666,
		Labels: map[string]string{"foo": "bar"},
	}

	pausingPm = MockedContainer{
		ID:     "10",
		Name:   "pausing_pm",
		Status: MockedPaused,
		PID:    10,
		Labels: map[string]string{"motto": "pifflepaffle"},
	}
)

var _ = Describe("mockingmoby", func() {

	It("looks up container by name or ID", func() {
		mm := NewMockingMoby()
		Expect(mm.DaemonHost()).NotTo(BeEmpty())

		defer mm.Close()
		mm.AddContainer(mockingMoby)

		_, ok := mm.lookup("foo")
		Expect(ok).To(BeFalse())

		c, ok := mm.lookup(mockingMoby.ID)
		Expect(ok).To(BeTrue())
		Expect(c).NotTo(BeNil())
		Expect(c.ID).To(Equal(mockingMoby.ID))

		c, ok = mm.lookup(mockingMoby.Name)
		Expect(ok).To(BeTrue())
		Expect(c).NotTo(BeNil())
		Expect(c.ID).To(Equal(mockingMoby.ID))
	})

})
