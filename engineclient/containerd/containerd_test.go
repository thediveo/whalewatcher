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

package containerd

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/thediveo/whalewatcher/engineclient"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

const testns = "whalewatcher-testing"

var _ = Describe("containerd engineclient", func() {

	It("generates container display IDs", func() {
		Expect(displayID("default", "foo")).To(Equal("foo"))
		Expect(displayID("rumpelpumpel", "foo")).To(Equal("rumpelpumpel/foo"))
	})

	It("regenerates container and namespace information from display IDs", func() {
		ns, id := decodeDisplayID("foo")
		Expect(ns).To(Equal("default"))
		Expect(id).To(Equal("foo"))

		ns, id = decodeDisplayID("rumpelpumpel/foo")
		Expect(ns).To(Equal("rumpelpumpel"))
		Expect(id).To(Equal("foo"))
	})

	It("watches...", func() {
		cwclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		cw := NewContainerdWatcher(cwclient)
		Expect(cw).NotTo(BeNil())
		defer cw.Close()

		ctx, cancel := context.WithCancel(context.Background())
		evs, errs := cw.LifecycleEvents(ctx)

		// https://containerd.io/docs/getting-started
		cdclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		defer cdclient.Close()
		wwctx := namespaces.WithNamespace(context.Background(), testns)

		// Clean up any trash left from a previously crashed/panic'ed unit
		// test...
		_, _ = cdclient.TaskService().Delete(wwctx, &tasks.DeleteTaskRequest{ContainerID: "buzzybocks"})
		_ = cdclient.ContainerService().Delete(wwctx, "buzzybocks")

		// Pull a busybox image, if not already locally available.
		busyboximg, err := cdclient.Pull(wwctx,
			"docker.io/library/busybox:latest", containerd.WithPullUnpack)
		Expect(err).NotTo(HaveOccurred())

		// Run a pausing test container by creating container+task, and finally
		// starting the task.
		buzzybocks, err := cdclient.NewContainer(wwctx,
			"buzzybocks",
			containerd.WithNewSnapshot("buzzybocks-snapshot", busyboximg),
			containerd.WithNewSpec(oci.WithImageConfigArgs(busyboximg,
				[]string{"/bin/sleep", "30s"})))
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			_ = buzzybocks.Delete(wwctx, containerd.WithSnapshotCleanup)
		}()
		buzzybockstask, err := buzzybocks.NewTask(wwctx, cio.NewCreator())
		Expect(err).NotTo(HaveOccurred())
		defer func() {
			_, _ = buzzybockstask.Delete(wwctx, containerd.WithProcessKill)
		}()
		err = buzzybockstask.Start(wwctx)
		Expect(err).NotTo(HaveOccurred())

		// We should see or have seen a task start event...
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Type": Equal(engineclient.ContainerStarted),
			"ID":   Equal(testns + "/buzzybocks"),
		})))

		// The container/task should also be listed...
		containers, err := cw.List(wwctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(containers).To(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"ID": Equal(testns + "/buzzybocks"),
		}))))

		// ...and we should be able to query its details.
		container, err := cw.Inspect(wwctx, testns+"/buzzybocks")
		Expect(err).NotTo(HaveOccurred())
		Expect(container.ID).To(Equal(testns + "/buzzybocks"))

		// Get rid of the task.
		_, err = buzzybockstask.Delete(wwctx, containerd.WithProcessKill)
		Expect(err).NotTo(HaveOccurred())

		// We should see or have seen the corresponding task exit event...
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Type": Equal(engineclient.ContainerExited),
			"ID":   Equal(testns + "/buzzybocks"),
		})))

		// Shut down the engine event stream and make sure that it closes the
		// error stream properly to signal its end...
		cancel()
		Eventually(errs).Should(BeClosed())
	})

})
