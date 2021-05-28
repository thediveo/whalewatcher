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
	"context"

	"github.com/thediveo/whalewatcher/test/mockingmoby"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	mocking_moby = mockingmoby.MockedContainer{
		ID:     "1234567890",
		Name:   "mocking_moby",
		Status: mockingmoby.MockedPaused,
		PID:    42,
		Labels: map[string]string{"motto": "I'm not dead yet"},
	}

	furious_furuncle = mockingmoby.MockedContainer{
		ID:     "6666666666",
		Name:   "furious_furuncle",
		Status: mockingmoby.MockedRunning,
		PID:    666,
		Labels: map[string]string{"foo": "bar"},
	}
)

var _ = Describe("whalewatcher", func() {

	It("adds newborn container to our portfolio", func() {
		mm := mockingmoby.NewMockingMoby()
		ww := NewWhalewatcher(mm)
		Expect(ww).NotTo(BeNil())
		mm.AddContainer(mocking_moby)

		ww.born(context.Background(), mocking_moby.ID)
		ww.list(context.Background())
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mocking_moby.Name))
	})

	It("removes dead container from our portfolio", func() {
		mm := mockingmoby.NewMockingMoby()
		ww := NewWhalewatcher(mm)
		Expect(ww).NotTo(BeNil())
		mm.AddContainer(mocking_moby)

		ww.born(context.Background(), mocking_moby.ID)
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mocking_moby.Name))

		ww.demised(mocking_moby.ID, "")
		Expect(ww.Portfolio().Project("").ContainerNames()).To(BeEmpty())
	})

	It("doesn't list zombies", func() {
		mm := mockingmoby.NewMockingMoby()
		ww := NewWhalewatcher(mm)
		Expect(ww).NotTo(BeNil())

		// Prime mocked moby and ensure that we find all containers in our
		// portfolio, so we know the simple case works.
		mm.AddContainer(mocking_moby)
		mm.AddContainer(furious_furuncle)
		ww.list(context.Background())
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mocking_moby.Name, furious_furuncle.Name))

		// Now check that containers dying while the list is in progress don't
		// get added to the portfolio, avoiding the portfolio getting filled
		// with zombies.
		ww.list(mockingmoby.WithHook(
			context.Background(),
			mockingmoby.ContainerListPost,
			func(mockingmoby.HookKey) error {
				mm.RemoveContainer(furious_furuncle.Name)
				ww.demised(furious_furuncle.ID, "")
				return nil
			}))
		Expect(ww.Portfolio().Project("").ContainerNames()).To(ConsistOf(mocking_moby.Name))
	})

	It("doesn't crash for failed list", func() {
		mm := mockingmoby.NewMockingMoby()
		ww := NewWhalewatcher(mm)
		Expect(ww).NotTo(BeNil())
		mm.AddContainer(mocking_moby)

		cctx, cancel := context.WithCancel(context.Background())
		cancel()
		ww.list(cctx)
		Expect(ww.Portfolio().Project("").ContainerNames()).To(BeEmpty())
	})

	It("binge watches", func() {
		mm := mockingmoby.NewMockingMoby()
		ww := NewWhalewatcher(mm)
		Expect(ww).NotTo(BeNil())
		mm.AddContainer(mocking_moby)

		cctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			ww.Watch(cctx)
			close(done)
		}()

		portfolio := func() []string {
			return ww.Portfolio().Project("").ContainerNames()
		}
		Eventually(portfolio).Should(ConsistOf(mocking_moby.Name))

		mm.AddContainer(furious_furuncle)
		Eventually(portfolio).Should(ConsistOf(mocking_moby.Name, furious_furuncle.Name))

		mm.RemoveContainer(furious_furuncle.ID)
		Eventually(portfolio).Should(ConsistOf(mocking_moby.Name))

		cancel()
		Eventually(done).Should(BeClosed())
	})

	It("resynchronizes", func() {
		mm := mockingmoby.NewMockingMoby()
		ww := NewWhalewatcher(mm)
		Expect(ww).NotTo(BeNil())
		portfolio := func() []string {
			return ww.Portfolio().Project("").ContainerNames()
		}
		mm.AddContainer(mocking_moby)

		cctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			ww.Watch(cctx)
			close(done)
		}()
		// Make sure that the watcher goroutine has properly started the event
		// streaming...
		Eventually(portfolio).Should(ConsistOf(mocking_moby.Name))

		// ...before stopping events. Otherwise: nice safeguard panic (instead
		// of deadlock).
		mm.StopEvents()
		Consistently(portfolio, "2s", "10ms").Should(ConsistOf(mocking_moby.Name))

		mm.AddContainer(furious_furuncle)
		Eventually(portfolio).Should(ConsistOf(mocking_moby.Name, furious_furuncle.Name))

		cancel()
		Eventually(done).Should(BeClosed())
	})

})
