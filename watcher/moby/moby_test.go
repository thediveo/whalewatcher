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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/ory/dockertest/v3"
	"github.com/thediveo/whalewatcher/engineclient/moby"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var slowSpec = NodeTimeout(20 * time.Second)

var _ = Describe("Moby watcher engine end-to-end test", func() {

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
		// to run any API additional API calls from the same goroutine as where
		// we start the Watch in order to not trigger the race detector.
		nchan := make(chan []types.NetworkResource, 1)
		go func() {
			defer GinkgoRecover()
			dc, ok := mw.Client().(client.APIClient)
			Expect(ok).To(BeTrue())
			Expect(dc).NotTo(BeNil())
			networks := Successful(dc.NetworkList(ctx, types.NetworkListOptions{}))
			nchan <- networks
			mw.Client().(client.CommonAPIClient).NegotiateAPIVersion(ctx)
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

		ctx, cancel := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() {
			_ = mw.Watch(ctx)
			close(done)
		}()
		Consistently(done, "1s").ShouldNot(BeClosed())

		pool := Successful(dockertest.NewPool("unix:///var/run/docker.sock"))
		cntr := Successful(pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "busybox",
			// ...here, we don't care about the name here, as long as we get a
			// fresh container.
			Tag: "latest",
			Cmd: []string{"/bin/sleep", "30s"},
			Labels: map[string]string{
				"com.docker.compose.project": "whalewatcher_whackywhale",
			},
		}))
		var purge sync.Once
		defer purge.Do(func() { _ = pool.Purge(cntr) })

		// eventually there should be a container poping up with the correct
		// composer project label.
		portfolio := func() []string {
			if proj := mw.Portfolio().Project("whalewatcher_whackywhale"); proj != nil {
				return proj.ContainerNames()
			}
			return []string{}
		}
		Eventually(portfolio).Should(ConsistOf(cntr.Container.Name[1:]))

		// and envtually that container should also be gone from the watch list
		// after we killed it.
		purge.Do(func() {
			Expect(pool.Purge(cntr)).To(Succeed())
		})
		Eventually(portfolio).Should(BeEmpty())

		// wait for the watcher to correctly spin down.
		cancel()
		Eventually(done).Should(BeClosed())
	})

})
