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
	"context"

	"github.com/thediveo/whalewatcher/engineclient"
	"github.com/thediveo/whalewatcher/test/mockingmoby"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/noleak"
	. "github.com/thediveo/whalewatcher/test/matcher"
)

var (
	furiousFuruncle = mockingmoby.MockedContainer{
		ID:     "6666666666",
		Name:   "furious_furuncle",
		Status: mockingmoby.MockedRunning,
		PID:    666,
		Labels: map[string]string{ComposerProjectLabel: "testproject"},
	}

	deadDummy = mockingmoby.MockedContainer{
		ID:     "1234567890",
		Name:   "dead_dummy",
		Status: mockingmoby.MockedDead,
	}

	madMay = mockingmoby.MockedContainer{
		ID:     "1234567890",
		Name:   "mad_mary",
		Status: mockingmoby.MockedRunning,
		PID:    666666,
		Labels: map[string]string{ComposerProjectLabel: "testproject"},
	}
)

var _ = Describe("moby engineclient", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Goroutines).ShouldNot(HaveLeaked())
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	var mm *mockingmoby.MockingMoby
	var ec *MobyWatcher

	BeforeEach(func() {
		mm = mockingmoby.NewMockingMoby()
		ec = NewMobyWatcher(mm, WithPID(123456))
		Expect(ec.PID()).To(Equal(123456))
		mm.AddContainer(furiousFuruncle)
	})

	AfterEach(func() {
		ec.Close()
	})

	It("has engine type ID and API path", func() {
		Expect(ec.Type()).To(Equal(Type))
		Expect(ec.API()).NotTo(BeEmpty())
	})

	It("has an ID and version", func() {
		ctx, cancel := context.WithCancel(context.Background())
		Expect(ec.ID(ctx)).ToNot(BeEmpty())
		Expect(ec.Version(ctx)).NotTo(BeEmpty())
		cancel()
		Expect(ec.ID(ctx)).To(BeZero())
	})

	It("cannot inspect a dead container", func() {
		mm.AddContainer(deadDummy)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		_, err := ec.Inspect(ctx, deadDummy.ID)
		Expect(err).To(HaveOccurred())
		Expect(engineclient.IsProcesslessContainer(err)).To(BeTrue())
	})

	It("inspects a furuncle", func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cntr, err := ec.Inspect(ctx, furiousFuruncle.ID)
		Expect(err).NotTo(HaveOccurred())
		Expect(cntr).To(HaveID(furiousFuruncle.ID))

		mm.AddContainer(deadDummy)
		_, err = ec.Inspect(ctx, deadDummy.ID)
		Expect(err).To(MatchError(MatchRegexp(`no initial process`)))

		cancel()
		_, err = ec.Inspect(ctx, furiousFuruncle.ID)
		Expect(err).To(HaveOccurred())
	})

	It("lists furuncle", func() {
		ctx, cancel := context.WithCancel(context.Background())

		cntr, err := ec.List(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(cntr).To(ConsistOf(HaveID(furiousFuruncle.ID)))

		cancel()
		_, err = ec.List(ctx)
		Expect(err).To(HaveOccurred())
	})

	It("watches containers come and go", func() {
		ctx, cancel := context.WithCancel(context.Background())

		evs, errs := ec.LifecycleEvents(ctx)
		Expect(evs).NotTo(BeNil())
		Expect(errs).NotTo(BeNil())

		Consistently(evs).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())

		By("adding a new container")
		mm.AddContainer(madMay)
		Eventually(evs).Should(Receive(And(
			HaveID(madMay.ID),
			HaveEventType(engineclient.ContainerStarted),
			HaveProject(madMay.Labels[ComposerProjectLabel]),
		)))

		By("pausing the container")
		mm.PauseContainer(madMay.ID)
		Eventually(evs).Should(Receive(And(
			HaveID(madMay.ID),
			HaveEventType(engineclient.ContainerPaused),
			HaveProject(madMay.Labels[ComposerProjectLabel]),
		)))

		By("unpausing the container")
		mm.UnpauseContainer(madMay.ID)
		Eventually(evs).Should(Receive(And(
			HaveID(madMay.ID),
			HaveEventType(engineclient.ContainerUnpaused),
			HaveProject(madMay.Labels[ComposerProjectLabel]),
		)))

		By("removing the container")
		mm.RemoveContainer(madMay.ID)
		Eventually(evs).Should(Receive(And(
			HaveID(madMay.ID),
			HaveEventType(engineclient.ContainerExited),
			HaveProject(madMay.Labels[ComposerProjectLabel]),
		)))

		cancel()
		Eventually(errs).Should(Receive(Equal(ctx.Err())))
	})

})
