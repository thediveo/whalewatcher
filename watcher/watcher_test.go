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
	"errors"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/thediveo/whalewatcher/engineclient"
	"github.com/thediveo/whalewatcher/engineclient/moby"
	"github.com/thediveo/whalewatcher/test/mockingmoby"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
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

type trialEngine struct {
	engineclient.EngineClient

	Retries uint32 // the first number of trials to fail
	trials  uint32 // number of backoff trials
}

// Trials returns the number of backoff trails so far.
func (e *trialEngine) Trials() uint32 {
	return atomic.LoadUint32(&e.trials)
}

var _ engineclient.Trialer = (*trialEngine)(nil)

// ClearTrials resets the trial counter.
func (e *trialEngine) ClearTrials() {
	atomic.StoreUint32(&e.trials, 0)
}

// Try counts the number of trials and returns an error for the first (number
// of) Retries.
func (e *trialEngine) Try(ctx context.Context) error {
	if atomic.AddUint32(&e.trials, 1) <= e.Retries {
		return errors.New("thou shall retry")
	}
	return nil
}

var _ = Describe("watcher (of whales, not: Wales)", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Goroutines).ShouldNot(HaveLeaked())
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	var mm *mockingmoby.MockingMoby
	var te *trialEngine
	var ww *watcher

	BeforeEach(func() {
		mm = mockingmoby.NewMockingMoby()
		Expect(mm).NotTo(BeNil())
		te = &trialEngine{
			EngineClient: moby.NewMobyWatcher(mm),
		}
		ww = New(te, backoff.NewConstantBackOff(500*time.Millisecond)).(*watcher)
		Expect(ww).NotTo(BeNil())

		DeferCleanup(func() {
			ww.Close()
		})
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

		evs := ww.Events()
		ww.born(context.Background(), mockingMoby.ID)
		Eventually(evs).Should(Receive(And(
			HaveField("Type", engineclient.ContainerStarted),
			HaveField("Container", Not(BeNil())),
		)))
		Expect(ww.list(context.Background())).To(Succeed())
		Consistently(evs).WithTimeout(2 * time.Second).WithPolling(250 * time.Millisecond).ShouldNot(Receive())
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mockingMoby.Name))
		ww.Close()
		Eventually(evs).Should(BeClosed())
	})

	It("adds newborn project container to our portfolio", func() {
		mm.AddContainer(porosePorpoise)

		ww.born(context.Background(), mockingMoby.ID)
		Expect(ww.list(context.Background())).To(Succeed())
		Expect(ww.Portfolio().Project("porose").ContainerNames()).To(ConsistOf(porosePorpoise.Name))
	})

	It("removes dead container from our portfolio", func() {
		mm.AddContainer(mockingMoby)

		evs := ww.Events()
		ww.born(context.Background(), mockingMoby.ID)
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mockingMoby.Name))

		ww.demised(mockingMoby.ID, "")
		Eventually(evs).Should(Receive(And(
			HaveField("Type", engineclient.ContainerExited),
			HaveField("Container", Not(BeNil())),
		)))
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
		evs := ww.Events()
		Expect(ww.list(mockingmoby.WithHook(
			context.Background(),
			mockingmoby.ContainerListPost,
			func(mockingmoby.HookKey) error {
				mm.RemoveContainer(furiousFuruncle.Name)
				ww.demised(furiousFuruncle.ID, "")
				return nil
			}))).To(Succeed())
		Eventually(evs).Should(Receive(And(
			HaveField("Type", engineclient.ContainerExited),
			HaveField("Container", Not(BeNil())),
		)))
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mockingMoby.Name))
	})

	It("correctly states pausing state while listing", func() {
		// Prime mocked moby and ensure that we find all containers in our
		// portfolio, so we know the simple case works.
		mm.AddContainer(mockingMoby)
		mm.AddContainer(furiousFuruncle)

		// (un)pause events during a list should be queued and properly handled
		// later.
		evs := ww.Events()
		Expect(ww.list(mockingmoby.WithHook(
			context.Background(),
			mockingmoby.ContainerListPost,
			func(mockingmoby.HookKey) error {
				mm.PauseContainer(furiousFuruncle.ID)
				ww.paused(furiousFuruncle.ID, "", true)
				return nil
			}))).To(Succeed())
		Eventually(evs).Should(Receive(And(
			HaveField("Type", engineclient.ContainerPaused),
			HaveField("Container", Not(BeNil())),
		)))
		Expect(ww.Portfolio().Project("").Container(furiousFuruncle.ID).Paused).To(BeTrue())

		// a later unpause should be propagate "immediately".
		mm.PauseContainer(furiousFuruncle.ID)
		ww.paused(furiousFuruncle.ID, "", false)
		Eventually(evs).Should(Receive(And(
			HaveField("Type", engineclient.ContainerUnpaused),
			HaveField("Container", Not(BeNil())),
		)))
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

		te.ClearTrials()
		te.Retries = 2
		go func() {
			_ = ww.Watch(cctx)
			close(done)
		}()

		// wait for the initial synchronization to be done and the initial
		// discovery results having just come in. As we're doing retries with
		// pauses in between, we must have an initial period where the
		// synchronization must not already be done.
		Consistently(ww.Ready()).WithTimeout(1 * time.Second).ShouldNot(BeClosed())
		Eventually(ww.Ready()).WithTimeout(2 * time.Second).Should(BeClosed())

		// Check that retries were done.
		Expect(te.Trials()).To(Equal(te.Retries + 1))

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
