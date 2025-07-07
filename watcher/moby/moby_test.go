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
	"sync"
	"time"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/thediveo/morbyd"
	"github.com/thediveo/morbyd/run"
	"github.com/thediveo/morbyd/session"
	"github.com/thediveo/whalewatcher/engineclient/moby"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var slowSpec = NodeTimeout(30 * time.Second)

var _ = Describe("Moby engine watcher end-to-end test", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Goroutines).ShouldNot(HaveLeaked())
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	It("doesn't accept invalid engine API paths", func() {
		Expect(New("localhost:66666", nil)).Error().To(HaveOccurred())
	})

	It("gets and uses the underlying Docker client", Serial, slowSpec, func(ctx context.Context) {
		mw := Successful(New("unix:///var/run/docker.sock", nil, moby.WithPID(123456)))

		Expect(mw.PID()).To(Equal(123456))
		defer mw.Close()

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		done := make(chan struct{})
		// While // https://github.com/moby/moby/pull/42379 is pending we need
		// to run any additional API calls from the same goroutine as where we
		// start the Watch in order to not trigger the race detector.
		nchan := make(chan []network.Summary, 1)
		go func() {
			defer GinkgoRecover()
			dc, ok := mw.Client().(client.APIClient)
			Expect(ok).To(BeTrue())
			Expect(dc).NotTo(BeNil())
			networks := Successful(dc.NetworkList(ctx, network.ListOptions{}))
			nchan <- networks
			mw.Client().(client.APIClient).NegotiateAPIVersion(ctx)
			_ = mw.Watch(ctx)
			close(done)
		}()
		Consistently(done).WithTimeout(5 * time.Second).WithPolling(250 * time.Millisecond).
			ShouldNot(BeClosed())
		networks := <-nchan
		Expect(networks).To(ContainElement(And(
			HaveField("Name", Equal("bridge")),
			HaveField("Driver", Equal("bridge")),
		)))
	})

	It("watches", Serial, slowSpec, func(ctx context.Context) {
		mw := Successful(New("unix:///var/run/docker.sock", nil, moby.WithPID(123456)))
		Expect(mw.PID()).To(Equal(123456))
		defer mw.Close()

		wctx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			_ = mw.Watch(wctx)
			close(done)
		}()
		Consistently(done, "1s").ShouldNot(BeClosed())

		By("creating a new Docker session for testing")
		sess := Successful(morbyd.NewSession(ctx,
			session.WithAutoCleaning("test.whalewatcher=watcher/moby")))
		DeferCleanup(func(ctx context.Context) {
			By("auto-cleaning the session")
			sess.Close(ctx)
		})
		cntr := Successful(sess.Run(ctx, "busybox",
			run.WithAutoRemove(),
			run.WithCommand("/bin/sh", "-c", "while true; do sleep 1; done"),
			run.WithLabel("com.docker.compose.project=whalewatcher_whackywhale")))

		var purge sync.Once
		defer purge.Do(func() { cntr.Kill(ctx) })

		// eventually there should be a container poping up with the correct
		// composer project label.
		portfolio := func() []string {
			if proj := mw.Portfolio().Project("whalewatcher_whackywhale"); proj != nil {
				return proj.ContainerNames()
			}
			return []string{}
		}
		Eventually(portfolio).Should(ConsistOf(cntr.Name))

		// and eventually that container should also be gone from the watch list
		// after we killed it.
		purge.Do(func() {
			cntr.Kill(ctx)
		})
		Eventually(portfolio).Should(BeEmpty())

		// wait for the watcher to correctly spin down.
		cancel()
		Eventually(done).Should(BeClosed())
	})

})
