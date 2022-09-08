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
	"net/http"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	"github.com/containerd/containerd/oci"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/thediveo/whalewatcher"
	"github.com/thediveo/whalewatcher/engineclient"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/whalewatcher/test/matcher"
)

type packer struct{}

func (p *packer) Pack(container *whalewatcher.Container, inspection interface{}) {
	Expect(container).NotTo(BeNil())
	Expect(inspection).NotTo(BeNil())
	var details InspectionDetails
	Expect(inspection).To(BeAssignableToTypeOf(details))
	details = inspection.(InspectionDetails)
	Expect(details.Container).NotTo(BeNil())
	Expect(details.Process).NotTo(BeNil())
	container.Rucksack = &details
}

var _ = Describe("containerd engineclient", func() {

	BeforeEach(func() {
		goodfds := Filedescriptors()
		DeferCleanup(func() {
			Eventually(Goroutines).ShouldNot(HaveLeaked())
			Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
		})
	})

	It("containerd client doesn't leak goroutines", func() {
		if os.Getegid() != 0 {
			Skip("needs root")
		}
		cdclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		Expect(cdclient.Server(context.Background())).Error().NotTo(HaveOccurred())
		defer cdclient.Close()
	})

	It("has engine ID and version", func() {
		if os.Getegid() != 0 {
			Skip("needs root")
		}
		cwclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		cw := NewContainerdWatcher(cwclient, WithPID(123456))
		Expect(cw).NotTo(BeNil())
		defer cw.Close()

		Expect(cw.PID()).To(Equal(123456))
		Expect(cw.ID(context.Background())).NotTo(BeEmpty())
		Expect(cw.Version(context.Background())).NotTo(BeEmpty())
	})

	It("survives cancelled contexts", func() {
		if os.Getegid() != 0 {
			Skip("needs root")
		}
		cwclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		cw := NewContainerdWatcher(cwclient)
		Expect(cw).NotTo(BeNil())
		defer cw.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		cntrs, err := cw.List(ctx)
		Expect(err).To(HaveOccurred())
		Expect(cntrs).To(BeNil())

		cntr, err := cw.Inspect(ctx, "never_ever_existing_foobar")
		Expect(err).To(HaveOccurred())
		Expect(cntr).To(BeNil())
	})

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

	It("sets a rucksack packer", func() {
		if os.Getegid() != 0 {
			Skip("needs root")
		}
		cwclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		p := packer{}
		cw := NewContainerdWatcher(cwclient, WithRucksackPacker(&p))
		Expect(cw).NotTo(BeNil())
		defer cw.Close()
		Expect(cw.packer).To(BeIdenticalTo(&p))
	})

	It("returns the underlying client", func() {
		if os.Getegid() != 0 {
			Skip("needs root")
		}
		cwclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		cw := NewContainerdWatcher(cwclient)
		Expect(cw).NotTo(BeNil())
		defer cw.Close()
		Expect(cw.Client()).To(BeIdenticalTo(cw.client))
	})

	It("watches...", Serial, func() {
		if os.Getegid() != 0 {
			Skip("needs root")
		}

		const bibi = "buzzybocks"
		const testns = "whalewatcher-testing"

		By("watching containerd engine")
		cwclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		cw := NewContainerdWatcher(cwclient)
		Expect(cw).NotTo(BeNil())
		defer cw.Close()

		Expect(cw.Type()).To(Equal(Type))
		Expect(cw.API()).NotTo(BeEmpty())

		ctx, cancel := context.WithCancel(context.Background())
		evs, errs := cw.LifecycleEvents(ctx)

		// https://containerd.io/docs/getting-started
		cdclient, err := containerd.New("/run/containerd/containerd.sock")
		Expect(err).NotTo(HaveOccurred())
		defer cdclient.Close()
		wwctx := namespaces.WithNamespace(ctx, testns)

		// Clean up any trash left from a previously crashed/panic'ed unit
		// test...
		_, _ = cdclient.TaskService().Delete(wwctx, &tasks.DeleteTaskRequest{ContainerID: bibi})
		_ = cdclient.ContainerService().Delete(wwctx, bibi)
		_ = cdclient.SnapshotService("").Remove(wwctx, bibi+"-snapshot")

		By("pulling a busybox image (if not already available locally)")
		// OUCH! the default resolver keeps a persistent client around, so this
		// triggers the goroutine leak detection. Thus, we need to explicitly
		// supply our own (default) client, which we have control over. In
		// particular, we can close its idle connections at the end of the test,
		// getting rid of idling persistent connections. Sigh.
		httpclnt := &http.Client{}
		busyboximg, err := cdclient.Pull(wwctx,
			"docker.io/library/busybox:latest",
			containerd.WithPullUnpack,
			containerd.WithResolver(docker.NewResolver(docker.ResolverOptions{
				Client: httpclnt,
			})))
		Expect(err).NotTo(HaveOccurred())
		defer httpclnt.CloseIdleConnections()

		By("creating a new container/task and starting it")
		// Run a pausing test container by creating container+task, and finally
		// starting the task.
		buzzybocks, err := cdclient.NewContainer(wwctx,
			bibi,
			containerd.WithNewSnapshot(bibi+"-snapshot", busyboximg),
			containerd.WithNewSpec(oci.WithImageConfigArgs(busyboximg,
				[]string{"/bin/sleep", "30s"})),
			containerd.WithAdditionalContainerLabels(map[string]string{
				"foo":            "bar",
				NerdctlNameLabel: "rappelfatz",
			}))
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

		By("receiving the newly started container/task event")
		Eventually(evs).Should(Receive(And(
			HaveEventType(engineclient.ContainerStarted),
			HaveID(testns+"/"+bibi),
		)))

		By("listing the newly started container/task")
		// The container/task should also be listed...
		containers, err := cw.List(wwctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(containers).To(ContainElement(HaveValue(And(
			HaveID(testns+"/"+bibi),
			HaveName(testns+"/rappelfatz"),
		))))

		By("getting details of the newly started container/task")
		// ...and we should be able to query its details.
		defer func() { cw.packer = nil }()
		cw.packer = &packer{}
		container, err := cw.Inspect(wwctx, testns+"/"+bibi)
		Expect(err).NotTo(HaveOccurred())
		Expect(container).To(HaveValue(And(
			HaveID(testns+"/"+bibi),
			HaveName(testns+"/rappelfatz"),
		)))
		Expect(container.Rucksack).NotTo(BeNil())

		By("pausing container/task")
		// pause...
		Expect(buzzybockstask.Pause(wwctx)).NotTo(HaveOccurred())
		Eventually(evs).Should(Receive(And(
			HaveEventType(engineclient.ContainerPaused),
			HaveID(testns+"/"+bibi),
		)))

		By("unpausing container/task")
		// ...and unpause it.
		Expect(buzzybockstask.Resume(wwctx)).NotTo(HaveOccurred())
		Eventually(evs).Should(Receive(And(
			HaveEventType(engineclient.ContainerUnpaused),
			HaveID(testns+"/"+bibi),
		)))

		By("deleting container/task")
		// Get rid of the task.
		_, err = buzzybockstask.Delete(wwctx, containerd.WithProcessKill)
		Expect(err).NotTo(HaveOccurred())

		By("receiving container/task exit event")
		// We should see or have seen the corresponding task exit event...
		Eventually(evs).Should(Receive(And(
			HaveEventType(engineclient.ContainerExited),
			HaveID(testns+"/"+bibi),
		)))

		By("closing down the event stream")
		// Shut down the engine event stream and make sure that it closes the
		// error stream properly to signal its end...
		cancel()
		Eventually(errs).Should(BeClosed())
	})

	Context("dynamic container workload", Ordered, func() {

		It("ignores Docker containers at containerd level", func() {
			if os.Getegid() != 0 {
				Skip("needs root")
			}

			const mobyns = "moby"
			const momo = "morbid_moby"

			By("watching containerd engine")
			cwclient, err := containerd.New("/run/containerd/containerd.sock")
			Expect(err).NotTo(HaveOccurred())
			cw := NewContainerdWatcher(cwclient)
			Expect(cw).NotTo(BeNil())
			defer cw.Close()

			Expect(cw.Type()).To(Equal(Type))
			Expect(cw.API()).NotTo(BeEmpty())

			ctx, cancel := context.WithCancel(context.Background())
			evs, errs := cw.LifecycleEvents(ctx)

			wwctx := namespaces.WithNamespace(ctx, mobyns)

			// Clean up any trash left from a previously crashed/panic'ed unit
			// test...
			_, _ = cwclient.TaskService().Delete(wwctx, &tasks.DeleteTaskRequest{ContainerID: momo})
			_ = cwclient.ContainerService().Delete(wwctx, momo)
			_ = cwclient.SnapshotService("").Remove(wwctx, momo+"-snapshot")

			By("pulling a busybox image (if not already available locally)")
			// OUCH! the default resolver keeps a persistent client around, so this
			// triggers the goroutine leak detection. Thus, we need to explicitly
			// supply our own (default) client, which we have control over. In
			// particular, we can close its idle connections at the end of the test,
			// getting rid of idling persistent connections. Sigh.
			httpclnt := &http.Client{}
			busyboximg, err := cwclient.Pull(wwctx,
				"docker.io/library/busybox:latest",
				containerd.WithPullUnpack,
				containerd.WithResolver(docker.NewResolver(docker.ResolverOptions{
					Client: httpclnt,
				})))
			Expect(err).NotTo(HaveOccurred())
			defer httpclnt.CloseIdleConnections()

			By("creating a new container/task and starting it")
			// Run a test container by creating container+task, in Docker's moby
			// namespace.
			morbidmoby, err := cwclient.NewContainer(wwctx,
				momo,
				containerd.WithNewSnapshot(momo+"-snapshot", busyboximg),
				containerd.WithNewSpec(oci.WithImageConfigArgs(busyboximg,
					[]string{"/bin/sleep", "30s"})))
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_ = morbidmoby.Delete(wwctx, containerd.WithSnapshotCleanup)
			}()
			morbidmobystask, err := morbidmoby.NewTask(wwctx, cio.NewCreator())
			Expect(err).NotTo(HaveOccurred())
			defer func() {
				_, _ = morbidmobystask.Delete(wwctx, containerd.WithProcessKill)
			}()
			err = morbidmobystask.Start(wwctx)
			Expect(err).NotTo(HaveOccurred())

			// We should never see any event for Docker-originating containers.
			Eventually(evs).ShouldNot(Receive(HaveID(mobyns + "/" + momo)))

			By("not seeing the newly started container/task in moby namespace")
			// We must not see this started container, as it is in the blocked
			// "moby" namespace.
			cntrs, err := cw.List(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(cntrs).NotTo(ContainElement(HaveValue(HaveID(mobyns + "/" + momo))))

			By("deleting container/task")
			// Get rid of the task.
			_, err = morbidmobystask.Delete(wwctx, containerd.WithProcessKill)
			Expect(err).NotTo(HaveOccurred())

			By("receiving container/task exit event")
			// We should see or have seen the corresponding task exit event...
			Eventually(evs).ShouldNot(Receive(HaveID(mobyns + "/" + momo)))

			By("closing down the event stream")
			// Shut down the engine event stream and make sure that it closes the
			// error stream properly to signal its end...
			cancel()
			Eventually(errs).Should(BeClosed())
		})

	})

})
