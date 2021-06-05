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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/ory/dockertest"
)

var _ = Describe("Moby watcher engine end-to-end test", func() {

	It("doesn't accept invalid engine API paths", func() {
		_, err := NewWatcher("localhost:66666")
		Expect(err).To(HaveOccurred())
	})

	It("watches", func() {
		mw, err := NewWatcher("unix:///var/run/docker.sock")
		Expect(err).NotTo(HaveOccurred())
		defer mw.Close()

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() {
			mw.Watch(ctx)
			close(done)
		}()
		Consistently(done, "1s").ShouldNot(BeClosed())

		pool, err := dockertest.NewPool("unix:///var/run/docker.sock")
		Expect(err).NotTo(HaveOccurred())
		cntr, err := pool.RunWithOptions(&dockertest.RunOptions{
			Repository: "busybox",
			Tag:        "latest",
			Cmd:        []string{"/bin/sleep", "30s"},
			Labels: map[string]string{
				"com.docker.compose.project": "whackywhale",
			},
		})
		Expect(err).NotTo(HaveOccurred())
		var purge sync.Once
		defer purge.Do(func() { _ = pool.Purge(cntr) })

		portfolio := func() []string {
			if proj := mw.Portfolio().Project("whackywhale"); proj != nil {
				return proj.ContainerNames()
			}
			return []string{}
		}
		Eventually(portfolio).Should(ConsistOf(cntr.Container.Name[1:]))

		purge.Do(func() {
			Expect(pool.Purge(cntr)).NotTo(HaveOccurred())
		})
		Eventually(portfolio).Should(BeEmpty())

		cancel()
		Eventually(done).Should(BeClosed())
	})

})
