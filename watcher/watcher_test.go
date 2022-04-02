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

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/thediveo/whalewatcher/engineclient"
	"github.com/thediveo/whalewatcher/engineclient/moby"
	"github.com/thediveo/whalewatcher/test/mockingmoby"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/noleak"
)

var (
	mockingMoby = mockingmoby.MockedContainer{
		ID:     "1234567890",
		Name:   "mocking_moby",
		Status: mockingmoby.MockedPaused,
		PID:    42,
		Labels: map[string]string{"motto": "I'm not dead yet"},
	}

	furiousFuruncle = mockingmoby.MockedContainer{
		ID:     "6666666666",
		Name:   "furious_furuncle",
		Status: mockingmoby.MockedRunning,
		PID:    666,
		Labels: map[string]string{"foo": "bar"},
	}

	porosePorpoise = mockingmoby.MockedContainer{
		ID:     "deadbeefc01dcafe",
		Name:   "porose_porpoise",
		Status: mockingmoby.MockedRunning,
		PID:    12345,
		Labels: map[string]string{"com.docker.compose.project": "porose"},
	}
)

var _ = Describe("watcher (of whales, not: Wales)", func() {

	AfterEach(func() {
		Eventually(Goroutines).ShouldNot(HaveLeaked())
	})

	var mm *mockingmoby.MockingMoby
	var ww *watcher

	BeforeEach(func() {
		mm = mockingmoby.NewMockingMoby()
		Expect(mm).NotTo(BeNil())
		ww = New(moby.NewMobyWatcher(mm), backoff.NewConstantBackOff(500*time.Millisecond)).(*watcher)
		Expect(ww).NotTo(BeNil())
	})

	AfterEach(func() {
		ww.Close()
	})

	It("returns the engine ID and version", func() {
		Expect(ww.ID(context.Background())).NotTo(BeEmpty())
		Expect(ww.Version(context.Background())).NotTo(BeEmpty())
	})

	It("has type and API path", func() {
		Expect(ww.Type()).NotTo(BeEmpty())
		Expect(ww.API()).NotTo(BeEmpty())
	})

	It("adds newborn container to our portfolio", func() {
		mm.AddContainer(mockingMoby)

		ww.born(context.Background(), mockingMoby.ID)
		Expect(ww.list(context.Background())).To(Succeed())
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mockingMoby.Name))
	})

	It("adds newborn project container to our portfolio", func() {
		mm.AddContainer(porosePorpoise)

		ww.born(context.Background(), mockingMoby.ID)
		Expect(ww.list(context.Background())).To(Succeed())
		Expect(ww.Portfolio().Project("porose").ContainerNames()).To(ConsistOf(porosePorpoise.Name))
	})

	It("removes dead container from our portfolio", func() {
		mm.AddContainer(mockingMoby)

		ww.born(context.Background(), mockingMoby.ID)
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mockingMoby.Name))

		ww.demised(mockingMoby.ID, "")
		Expect(ww.Portfolio().Project("").ContainerNames()).To(BeEmpty())
	})

	It("removes dead project container from our portfolio", func() {
		mm.AddContainer(porosePorpoise)

		// Silently ignore events for non-existing container
		ww.demised("notorious_nirvana", engineclient.ProjectUnknown)

		ww.born(context.Background(), porosePorpoise.ID)
		Expect(ww.Portfolio().Project("porose").ContainerNames()).To(ConsistOf(porosePorpoise.Name))

		ww.demised(porosePorpoise.ID, engineclient.ProjectUnknown)
		Expect(ww.Portfolio().Project("porose")).To(BeNil())
	})

	It("doesn't list zombies", func() {
		// Prime mocked moby and ensure that we find all containers in our
		// portfolio, so we know the simple case works.
		mm.AddContainer(mockingMoby)
		mm.AddContainer(furiousFuruncle)
		Expect(ww.list(context.Background())).To(Succeed())
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mockingMoby.Name, furiousFuruncle.Name))

		// Now check that containers dying while the list is in progress don't
		// get added to the portfolio, avoiding the portfolio getting filled
		// with zombies.
		Expect(ww.list(mockingmoby.WithHook(
			context.Background(),
			mockingmoby.ContainerListPost,
			func(mockingmoby.HookKey) error {
				mm.RemoveContainer(furiousFuruncle.Name)
				ww.demised(furiousFuruncle.ID, "")
				return nil
			}))).To(Succeed())
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mockingMoby.Name))
	})

	It("correctly states pausing state while listing", func() {
		// Prime mocked moby and ensure that we find all containers in our
		// portfolio, so we know the simple case works.
		mm.AddContainer(mockingMoby)
		mm.AddContainer(furiousFuruncle)

		// (un)pause events during a list should be queued and properly handled
		// later.
		Expect(ww.list(mockingmoby.WithHook(
			context.Background(),
			mockingmoby.ContainerListPost,
			func(mockingmoby.HookKey) error {
				mm.PauseContainer(furiousFuruncle.ID)
				ww.paused(furiousFuruncle.ID, "", true)
				return nil
			}))).To(Succeed())
		Expect(ww.Portfolio().Project("").Container(furiousFuruncle.ID).Paused).To(BeTrue())

		// a later unpause should be propagate "immediately".
		mm.PauseContainer(furiousFuruncle.ID)
		ww.paused(furiousFuruncle.ID, "", false)
		Expect(ww.Portfolio().Project("").Container(furiousFuruncle.ID).Paused).To(BeFalse())
	})

	It("correctly drops states pausing state for dying container while listing", func() {
		// Prime mocked moby and ensure that we find all containers in our
		// portfolio, so we know the simple case works.
		mm.AddContainer(mockingMoby)
		mm.AddContainer(furiousFuruncle)

		// queued (un)pause state changes must be dropped for deleted container.
		Expect(ww.list(mockingmoby.WithHook(
			context.Background(),
			mockingmoby.ContainerListPost,
			func(mockingmoby.HookKey) error {
				mm.PauseContainer(furiousFuruncle.ID)
				ww.paused(furiousFuruncle.ID, "", true)
				mm.RemoveContainer(furiousFuruncle.ID)
				ww.demised(furiousFuruncle.ID, "")
				mm.AddContainer(furiousFuruncle)
				ww.born(context.Background(), furiousFuruncle.ID)
				return nil
			}))).To(Succeed())
		c := ww.Portfolio().Project("").Container(furiousFuruncle.ID)
		Expect(c).NotTo(BeNil())
		Expect(c.Paused).To(BeFalse())
	})

	It("correctly updates pausing state for resurrected container while listing", func() {
		// Prime mocked moby and ensure that we find all containers in our
		// portfolio, so we know the simple case works.
		mm.AddContainer(mockingMoby)
		mm.AddContainer(furiousFuruncle)

		// queued (un)pause state changes must be dropped for deleted container.
		Expect(ww.list(mockingmoby.WithHook(
			context.Background(),
			mockingmoby.ContainerListPost,
			func(mockingmoby.HookKey) error {
				mm.PauseContainer(furiousFuruncle.ID)
				ww.paused(furiousFuruncle.ID, "", true)
				mm.RemoveContainer(furiousFuruncle.ID)
				ww.demised(furiousFuruncle.ID, "")
				mm.AddContainer(furiousFuruncle)
				ww.born(context.Background(), furiousFuruncle.ID)
				ww.paused(furiousFuruncle.ID, "", true)
				mm.RemoveContainer(furiousFuruncle.ID)
				return nil
			}))).To(Succeed())
		c := ww.Portfolio().Project("").Container(furiousFuruncle.ID)
		Expect(c).NotTo(BeNil())
		Expect(c.Paused).To(BeTrue())
	})

	It("doesn't crash for failed list", func() {
		mm.AddContainer(mockingMoby)

		cctx, cancel := context.WithCancel(context.Background())
		cancel()
		Expect(ww.list(cctx)).To(MatchError(MatchRegexp(`context canceled`)))
		Expect(ww.Portfolio().Project("").ContainerNames()).To(BeEmpty())
	})

	It("binge watches", func() {
		mm.AddContainer(mockingMoby)

		cctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			_ = ww.Watch(cctx)
			close(done)
		}()

		// Pass ww.Ready, not its result: wait for the initial synchronization
		// to be done and the initial discovery results having just come in.
		Eventually(ww.Ready).Should(BeClosed())

		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mockingMoby.Name))

		portfolio := func() []string {
			return ww.Portfolio().Project("").ContainerNames()
		}

		mm.AddContainer(furiousFuruncle)
		Eventually(portfolio).Should(ConsistOf(mockingMoby.Name, furiousFuruncle.Name))

		ffpaused := func() bool {
			return ww.Portfolio().Project("").Container(furiousFuruncle.ID).Paused
		}
		mm.PauseContainer(furiousFuruncle.ID)
		Eventually(ffpaused).Should(BeTrue())

		mm.UnpauseContainer(furiousFuruncle.ID)
		Eventually(ffpaused).Should(BeFalse())

		mm.RemoveContainer(furiousFuruncle.ID)
		Eventually(portfolio).Should(ConsistOf(mockingMoby.Name))

		cancel()
		Eventually(done).Should(BeClosed())
	})

	It("resynchronizes (with backoff)", func() {
		portfolio := func() []string {
			return ww.Portfolio().Project("").ContainerNames()
		}
		mm.AddContainer(mockingMoby)

		cctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			_ = ww.Watch(cctx)
			close(done)
		}()
		// Make sure that the watcher goroutine has properly started the event
		// streaming...
		Eventually(portfolio).Should(ConsistOf(mockingMoby.Name))

		// ...before stopping events. Otherwise: nice safeguard panic (instead
		// of deadlock).
		mm.StopEvents() // triggers backoff with reconnect.
		Consistently(portfolio, "2s", "10ms").Should(ConsistOf(mockingMoby.Name))

		mm.AddContainer(furiousFuruncle)
		Eventually(portfolio).Should(ConsistOf(mockingMoby.Name, furiousFuruncle.Name))
		Expect(done).NotTo(BeClosed())

		cancel()
		Eventually(done).Should(BeClosed())
	})

	It("paused and unpauses project containers", func() {
		mm.AddContainer(porosePorpoise)

		// Silently ignore events for non-existing container
		ww.paused("notorious_nirvana", engineclient.ProjectUnknown, true)

		ww.born(context.Background(), porosePorpoise.ID)
		Expect(ww.Portfolio().Project("porose").ContainerNames()).To(ConsistOf(porosePorpoise.Name))

		ww.paused(porosePorpoise.ID, engineclient.ProjectUnknown, true)
		Expect(ww.Portfolio().Project("porose").Container(porosePorpoise.ID).Paused).To(BeTrue())

		ww.paused(porosePorpoise.ID, engineclient.ProjectUnknown, false)
		Expect(ww.Portfolio().Project("porose").Container(porosePorpoise.ID).Paused).To(BeFalse())
	})

})
