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

package matcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("All matcher", func() {

	It("succeeds for correct set of actuals", func() {
		actuals := []int{3, 42, 1}
		m := All(Equal(1), Equal(42), Equal(3))
		for idx, actual := range actuals {
			success, err := m.Match(actual)
			Expect(err).NotTo(HaveOccurred())
			if idx == len(actuals)-1 {
				Expect(success).To(BeTrue())
			} else {
				Expect(success).To(BeFalse())
			}
		}
	})

	It("errors for non-matching actual", func() {
		m := All(Equal(1), Equal(42), Equal(3))
		Expect(m.Match(3)).Error().NotTo(HaveOccurred())
		Expect(m.Match(666)).Error().To(HaveOccurred())
	})

})
