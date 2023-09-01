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
	"fmt"
	"io"
	"os"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/thediveo/whalewatcher"
	"github.com/thediveo/whalewatcher/engineclient"
	"github.com/thediveo/whalewatcher/engineclient/containerd/test/ctr"
	"github.com/thediveo/whalewatcher/engineclient/containerd/test/img"
	"github.com/thediveo/whalewatcher/test"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
	. "github.com/thediveo/whalewatcher/test/matcher"
)

const (
	slowSpec = NodeTimeout(20 * time.Second)

	// name of Docker container with containerd and ctr
	kindischName = "ww-engineclient-containerd"

	testNamespace     = "whalewatcher-testing"
	testContainerName = "buzzybocks"
	testImageRef      = "docker.io/library/busybox:latest"
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

var _ = Describe("containerd engineclient", Ordered, func() {

	Context("display ID translations", func() {

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

	})

	Context("using real engine", func() {

		var endpointPath string
		var providerCntr *dockertest.Resource

		BeforeAll(func(ctx context.Context) {
			if os.Getuid() != 0 {
				Skip("needs root")
			}

			// Make sure to also leak-check the overall setup and teardown and
			// not just the individual tests. In particular, this also checks
			// that the containerd client doesn't leak go routines.
			goodfds := Filedescriptors()
			DeferCleanup(func() {
				Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(250 * time.Millisecond).
					ShouldNot(HaveLeaked())
				Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
			})

			By("spinning up a Docker container with stand-alone containerd, courtesy of the KinD k8s sig")
			pool := Successful(dockertest.NewPool("unix:///var/run/docker.sock"))
			_ = pool.RemoveContainerByName(kindischName)
			// The necessary container start arguments come from KinD's Docker node
			// provisioner, see:
			// https://github.com/kubernetes-sigs/kind/blob/3610f606516ccaa88aa098465d8c13af70937050/pkg/cluster/internal/providers/docker/provision.go#L133
			//
			// Please note that --privileged already implies switching off AppArmor.
			//
			// Please note further, that currently some Docker client CLI flags
			// don't translate into dockertest-supported options.
			//
			// docker run -it --rm --name kindisch-...
			//   --privileged
			//   --cgroupns=private
			//   --init=false
			//   --volume /dev/mapper:/dev/mapper
			//   --device /dev/fuse
			//   --tmpfs /tmp
			//   --tmpfs /run
			//   --volume /var
			//   --volume /lib/modules:/lib/modules:ro
			//   kindisch-...
			Expect(pool.Client.BuildImage(docker.BuildImageOptions{
				Name:       img.Name,
				ContextDir: "./test/_kindisch", // sorry, couldn't resist the pun.
				Dockerfile: "Dockerfile",
				BuildArgs: []docker.BuildArg{
					{Name: "KINDEST_BASE_TAG", Value: test.KindestBaseImageTag},
				},
				OutputStream: io.Discard,
			})).To(Succeed())
			providerCntr = Successful(pool.RunWithOptions(
				&dockertest.RunOptions{
					Name:       kindischName,
					Repository: img.Name,
					Privileged: true,
					Mounts: []string{
						"/var", // well, this actually is an unnamed volume
						"/dev/mapper:/dev/mapper",
						"/lib/modules:/lib/modules:ro",
					},
					Tty: true,
				}, func(hc *docker.HostConfig) {
					hc.Init = false
					hc.Tmpfs = map[string]string{
						"/tmp": "",
						"/run": "",
					}
					hc.Devices = []docker.Device{
						{PathOnHost: "/dev/fuse"},
					}
				}))
			DeferCleanup(func() {
				By("removing the containerd Docker container")
				Expect(pool.Purge(providerCntr)).To(Succeed())
			})

			By("waiting for containerized containerd to become responsive")
			Expect(providerCntr.Container.State.Pid).NotTo(BeZero())
			// apipath must not include absolute symbolic links, but already be
			// properly resolved.
			endpointPath = fmt.Sprintf("/proc/%d/root%s",
				providerCntr.Container.State.Pid, "/run/containerd/containerd.sock")
			var cdclient *containerd.Client
			Eventually(func() error {
				var err error
				cdclient, err = containerd.New(endpointPath,
					containerd.WithTimeout(5*time.Second))
				return err
			}).Within(30*time.Second).ProbeEvery(1*time.Second).
				Should(Succeed(), "containerd API never became responsive")
			cdclient.Close() // not needed anymore, will create fresh ones over and over again
		})

		var cdclient *containerd.Client

		BeforeEach(func() {
			goodfds := Filedescriptors()
			DeferCleanup(func() {
				Eventually(Goroutines).Within(2 * time.Second).ProbeEvery(250 * time.Millisecond).
					ShouldNot(HaveLeaked())
				Expect(Filedescriptors()).NotTo(HaveLeakedFds(goodfds))
			})

			cdclient = Successful(containerd.New(endpointPath,
				containerd.WithTimeout(5*time.Second)))
		})

		It("has engine ID and version", func(ctx context.Context) {
			cw := NewContainerdWatcher(cdclient, WithPID(123456))
			Expect(cw).NotTo(BeNil())
			defer cw.Close()

			Expect(cw.PID()).To(Equal(123456))
			Expect(cw.ID(ctx)).NotTo(BeEmpty())
			Expect(cw.Version(ctx)).NotTo(BeEmpty())
		})

		It("correctly handles cancelled contexts", func(ctx context.Context) {
			cw := NewContainerdWatcher(cdclient)
			Expect(cw).NotTo(BeNil())
			defer cw.Close()

			ctx, cancel := context.WithCancel(ctx)
			cancel() // immediately cancel it to check error handling

			Expect(cw.List(ctx)).Error().To(HaveOccurred())
			Expect(cw.Inspect(ctx, "never_ever_existing_foobar")).Error().To(HaveOccurred())
		})

		It("sets a rucksack packer", func() {
			p := packer{}
			cw := NewContainerdWatcher(cdclient, WithRucksackPacker(&p))
			Expect(cw).NotTo(BeNil())
			defer cw.Close()
			Expect(cw.packer).To(BeIdenticalTo(&p))
		})

		It("returns the underlying client", func() {
			cw := NewContainerdWatcher(cdclient)
			Expect(cw).NotTo(BeNil())
			defer cw.Close()
			Expect(cw.Client()).To(BeIdenticalTo(cw.client))
		})

		It("watches the container workload...", slowSpec, func(ctx context.Context) {
			By("watching containerd engine")
			cw := NewContainerdWatcher(cdclient)
			Expect(cw).NotTo(BeNil())
			defer cw.Close()

			Expect(cw.Type()).To(Equal(Type))
			Expect(cw.API()).NotTo(BeEmpty())

			ctx, cancel := context.WithCancel(ctx)
			evs, errs := cw.LifecycleEvents(ctx)

			// https://containerd.io/docs/getting-started
			wwctx := namespaces.WithNamespace(ctx, testNamespace)

			By("pulling a busybox image (if necessary)")
			ctr.Successfully(providerCntr,
				"-n", testNamespace,
				"image", "pull", testImageRef)

			By("creating a new container+task and starting it")
			ctr.Successfully(providerCntr,
				"-n", testNamespace,
				"run", "-d",
				"--label", "foo=bar",
				"--label", NerdctlNameLabel+"=rappelfatz",
				testImageRef,
				testContainerName,
				"/bin/sleep", "30s")
			DeferCleanup(func() {
				_ = ctr.Exec(providerCntr,
					"-n", testNamespace,
					"task", "rm", "-f", testContainerName)
				_ = ctr.Exec(providerCntr,
					"-n", testNamespace,
					"container", "rm", testContainerName)
			})

			By("receiving the newly started container/task event")
			Eventually(evs).Should(Receive(And(
				HaveEventType(engineclient.ContainerStarted),
				HaveID(testNamespace+"/"+testContainerName),
			)))

			By("listing the newly started container/task")
			// The container/task should also be listed...
			containers := Successful(cw.List(wwctx))
			Expect(containers).To(ContainElement(HaveValue(And(
				HaveID(testNamespace+"/"+testContainerName),
				HaveName(testNamespace+"/rappelfatz"),
			))))

			By("getting details of the newly started container/task")
			// ...and we should be able to query its details.
			defer func() { cw.packer = nil }()
			cw.packer = &packer{}
			container := Successful(cw.Inspect(wwctx, testNamespace+"/"+testContainerName))
			Expect(container).To(HaveValue(And(
				HaveID(testNamespace+"/"+testContainerName),
				HaveName(testNamespace+"/rappelfatz"),
			)))
			Expect(container.Rucksack).NotTo(BeNil())

			By("pausing container/task")
			ctr.Successfully(providerCntr,
				"-n", testNamespace,
				"task", "pause", testContainerName)
			Eventually(evs).Should(Receive(And(
				HaveEventType(engineclient.ContainerPaused),
				HaveID(testNamespace+"/"+testContainerName),
			)))
			c := Successful(cw.Inspect(wwctx, testNamespace+"/"+testContainerName))
			Expect(c.Paused).To(BeTrue())

			By("unpausing container/task")
			ctr.Successfully(providerCntr,
				"-n", testNamespace,
				"task", "resume", testContainerName)
			Eventually(evs).Should(Receive(And(
				HaveEventType(engineclient.ContainerUnpaused),
				HaveID(testNamespace+"/"+testContainerName),
			)))
			c = Successful(cw.Inspect(wwctx, testNamespace+"/"+testContainerName))
			Expect(c.Paused).To(BeFalse())

			By("deleting container/task")
			ctr.Successfully(providerCntr,
				"-n", testNamespace,
				"task", "rm", "-f", testContainerName)

			By("receiving container/task exit event")
			// We should see or have seen the corresponding task exit event...
			Eventually(evs).Should(Receive(And(
				HaveEventType(engineclient.ContainerExited),
				HaveID(testNamespace+"/"+testContainerName),
			)))

			By("closing down the event stream")
			// Shut down the engine event stream and make sure that it closes the
			// error stream properly to signal its end...
			cancel()
			Eventually(errs).Should(BeClosed())
		})

		It("returns nil for a task-less container", func(ctx context.Context) {
			By("watching containerd engine")
			cw := NewContainerdWatcher(cdclient)
			Expect(cw).NotTo(BeNil())
			defer cw.Close()

			// https://containerd.io/docs/getting-started
			wwctx := namespaces.WithNamespace(ctx, testNamespace)

			By("pulling a busybox image (if necessary)")
			ctr.Successfully(providerCntr,
				"-n", testNamespace,
				"image", "pull", testImageRef)

			By("creating a new container, but not starting it")
			ctr.Successfully(providerCntr,
				"-n", testNamespace,
				"container", "create",
				"--label", "foo=bar",
				"--label", NerdctlNameLabel+"=rappelfatz",
				testImageRef,
				testContainerName,
				"/bin/sleep", "30s")

			DeferCleanup(func() {
				_ = ctr.Exec(providerCntr,
					"-n", testNamespace,
					"container", "rm", testContainerName)
			})

			Expect(cw.Inspect(wwctx, testNamespace+"/"+testContainerName)).Error().
				To(MatchError(MatchRegexp(`task .* not found`)))
		})

		Context("dynamic container workload", func() {

			It("ignores Docker containers at containerd level", func(ctx context.Context) {
				const mobyns = "moby"
				const momo = "morbid_moby"

				By("watching containerd engine")
				cw := NewContainerdWatcher(cdclient)
				Expect(cw).NotTo(BeNil())
				defer cw.Close()

				Expect(cw.Type()).To(Equal(Type))
				Expect(cw.API()).NotTo(BeEmpty())

				ctx, cancel := context.WithCancel(ctx)
				evs, errs := cw.LifecycleEvents(ctx)

				wwctx := namespaces.WithNamespace(ctx, mobyns)

				By("pulling a busybox image (if not already available locally)")
				ctr.Successfully(providerCntr,
					"-n", mobyns,
					"image", "pull", testImageRef)

				By("creating a new container+task and starting it")
				ctr.Successfully(providerCntr,
					"-n", mobyns,
					"run", "-d",
					"--label", "foo=bar",
					"--label", NerdctlNameLabel+"=rappelfatz",
					testImageRef,
					testContainerName,
					"/bin/sleep", "30s")
				DeferCleanup(func() {
					_ = ctr.Exec(providerCntr,
						"-n", mobyns,
						"task", "rm", "-f", testContainerName)
					_ = ctr.Exec(providerCntr,
						"-n", mobyns,
						"container", "rm", testContainerName)
				})

				// We should never see any event for Docker-originating containers.
				Eventually(evs).ShouldNot(Receive(HaveID(mobyns + "/" + momo)))

				By("not seeing the newly started container/task in moby namespace")
				// We must not see this started container, as it is in the blocked
				// "moby" namespace.
				cntrs := Successful(cw.List(ctx))
				Expect(cntrs).NotTo(ContainElement(HaveValue(HaveID(mobyns + "/" + momo))))

				By("stopping container/task")
				ctr.Successfully(providerCntr,
					"-n", mobyns,
					"task", "kill", "--signal", "9", testContainerName)
				Eventually(func() error {
					_, err := cw.Inspect(wwctx, mobyns+"/"+momo)
					return err
				}).Should(MatchError(MatchRegexp(`container .*: not found`)))

				By("deleting container/task")
				ctr.Successfully(providerCntr,
					"-n", mobyns,
					"task", "rm", "-f", testContainerName)

				By("not receiving container/task any exit event")
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

})
