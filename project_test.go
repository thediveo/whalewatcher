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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("composer project proxy", func() {

	It("prints", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())
		Expect(p.String()).To(Equal("empty composer project 'gnampf'"))

		p.add(&Container{Name: "furious_furuncle"})
		p.add(&Container{Name: "mad_moby"})
		Expect(p.String()).To(Equal(
			"composer project 'gnampf' with containers: 'furious_furuncle', 'mad_moby'"))
	})

	It("adds containers", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		ff := &Container{Name: "furious_furuncle"}
		p.add(ff)
		p.add(&Container{Name: "mad_moby"})

		Expect(p.ContainerNames()).To(ConsistOf(
			"furious_furuncle", "mad_moby"))
	})

	It("doesn't update an existing container", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		p.add(&Container{Name: "furious_furuncle", ID: "1"})
		p.add(&Container{Name: "mad_moby"})
		p.add(&Container{Name: "furious_furuncle", ID: "1"})

		Expect(p.Containers()).To(HaveLen(2))
		Expect(p.ContainerNames()).To(ConsistOf(
			"furious_furuncle", "mad_moby"))
		Expect(p.Containers()).To(ContainElement(HaveValue(And(
			HaveField("Name", Equal("furious_furuncle")),
			HaveField("ID", Equal("1")),
		))))
	})

	It("differentiates containers by name and ID", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		p.add(&Container{Name: "furious_furuncle", ID: "1"})
		p.add(&Container{Name: "furious_furuncle", ID: "2"})

		Expect(p.Containers()).To(HaveLen(2))
		Expect(p.ContainerNames()).To(ConsistOf(
			"furious_furuncle", "furious_furuncle"))
		Expect(p.Containers()).To(ContainElements(
			HaveValue(And(
				HaveField("Name", Equal("furious_furuncle")),
				HaveField("ID", Equal("1")),
			)),
			HaveValue(And(
				HaveField("Name", Equal("furious_furuncle")),
				HaveField("ID", Equal("2")),
			)),
		))
	})

	It("removes containers", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		ff := &Container{Name: "furious_furuncle"}
		p.add(ff)
		p.add(&Container{Name: "mad_moby"})

		p.remove("foobar")
		Expect(p.ContainerNames()).To(ConsistOf(
			"furious_furuncle", "mad_moby"))

		p.remove("furious_furuncle")
		Expect(p.ContainerNames()).To(ConsistOf("mad_moby"))

		p.remove("mad_moby")
		Expect(p.Containers()).To(BeEmpty())

		p.remove("mad_moby")
		Expect(p.Containers()).To(BeEmpty())
	})

	It("lists its containers", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		p.add(&Container{Name: "furious_furuncle"})
		cs := p.Containers()
		p.add(&Container{Name: "mad_moby"})
		Expect(p.Containers()).To(HaveLen(2))
		Expect(cs).To(HaveLen(1)) // Must not have changed.
	})

	It("finds a container", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		p.add(&Container{Name: "furious_furuncle"})
		p.add(&Container{Name: "mad_moby", ID: "666"})

		Expect(p.Container("rusty_rumpelpumpel")).To(BeNil())
		mm := p.Container("mad_moby")
		Expect(mm).NotTo(BeNil())
		Expect(mm.Name).To(Equal("mad_moby"))

		mm = p.Container("666")
		Expect(mm).NotTo(BeNil())
		Expect(mm.Name).To(Equal("mad_moby"))
	})

	It("updates a container's pause state", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		p.add(&Container{Name: "furious_furuncle"})
		ff := p.Container("furious_furuncle")
		Expect(ff.Paused).To(BeFalse())
		p.SetPaused("furious_furuncle", true)
		pff := p.Container("furious_furuncle")
		Expect(pff.Paused).To(BeTrue())
		Expect(ff.Paused).To(BeFalse())
	})

	It("ignores trying to pause a non-existing container", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		Expect(p.SetPaused("foobarz", true)).To(BeNil())
	})

	It("returns the original container when pause state is unchanged", func() {
		p := newComposerProject("gnampf")
		Expect(p).NotTo(BeNil())

		p.add(&Container{Name: "furious_furuncle"})
		ff := p.Container("furious_furuncle")
		ff2 := p.SetPaused(ff.Name, false)
		Expect(ff2).NotTo(BeNil())
		Expect(ff2).To(BeIdenticalTo(ff))
	})

})
