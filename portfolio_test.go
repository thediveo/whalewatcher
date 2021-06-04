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

package whalewatcher

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("composer project portfolio", func() {

	It("always has zero project", func() {
		pf := NewPortfolio()
		Expect(pf).NotTo(BeNil())
		Expect(pf.Names()).To(BeEmpty())
		Expect(pf.Project("")).NotTo(BeNil())
		Expect(pf.Project("grumpy")).To(BeNil())
		Expect(pf.ContainerTotal()).To(Equal(0))
	})

	It("adds containers and projects", func() {
		pf := NewPortfolio()
		Expect(pf).NotTo(BeNil())

		pf.Add(&Container{
			Name:    "furious_furuncle",
			Project: "grumpy",
		})
		pf.Add(&Container{
			Name:    "murky_moby",
			Project: "grumpy",
		})
		Expect(pf.Names()).To(ConsistOf("grumpy"))

		grumpy := pf.Project("grumpy")
		Expect(grumpy).NotTo(BeNil())
		Expect(grumpy.ContainerNames()).To(ConsistOf("murky_moby", "furious_furuncle"))
		Expect(pf.ContainerTotal()).To(Equal(2))
	})

	It("removes containers and projects", func() {
		pf := NewPortfolio()
		Expect(pf).NotTo(BeNil())

		pf.Add(&Container{
			Name:    "furious_furuncle",
			Project: "grumpy",
		})
		pf.Add(&Container{
			Name:    "murky_moby",
			Project: "grumpy",
		})
		Expect(pf.Names()).To(ConsistOf("grumpy"))

		Expect(pf.Project("grumpy")).NotTo(BeNil())

		pf.Remove("missing_moby", "")
		Expect(pf.Project("")).NotTo(BeNil())

		pf.Remove("missing_moby", "grumpy")
		Expect(pf.Project("grumpy").ContainerNames()).To(ConsistOf("murky_moby", "furious_furuncle"))

		pf.Remove("murky_moby", "grumpy")
		Expect(pf.Project("grumpy").ContainerNames()).To(ConsistOf("furious_furuncle"))

		pf.Remove("furious_furuncle", "grumpy")
		Expect(pf.Project("grumpy")).To(BeNil())
		Expect(pf.ContainerTotal()).To(Equal(0))
	})

})
