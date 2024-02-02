// Copyright 2023 Harald Albrecht.
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

package cri

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/thediveo/morbyd"
	"github.com/thediveo/morbyd/build"
	"github.com/thediveo/morbyd/run"
	"github.com/thediveo/morbyd/timestamper"
	criengine "github.com/thediveo/whalewatcher/engineclient/cri"
	"github.com/thediveo/whalewatcher/engineclient/cri/test/img"
	"github.com/thediveo/whalewatcher/test"
	"github.com/thediveo/whalewatcher/watcher"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gleak"
	. "github.com/thediveo/fdooze"
	. "github.com/thediveo/success"
)

var slowSpec = NodeTimeout(30 * time.Second)

const (
	// name of Docker container with containerd+cri-o; we actually only need containerd
	kindischName = "ww-watcher-cri"

	k8sTestNamespace = "wwcriwwtest"
	k8sTestPodName   = "wwcritestpod"
)

var _ = Describe("CRI watcher engine end-to-end test", Ordered, Serial, func() {

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

	Context("containerized CRI engine", Ordered, func() {

		var providerCntr *morbyd.Container

		// We build and use the same Docker container for testing our CRI event API
		// client with both containerd as well as cri-o. Fortunately, installing
		// cri-o on top of the containerd-powered kindest/base image turns out to be
		// not that complicated.
		BeforeAll(func(ctx context.Context) {
			if os.Getuid() != 0 {
				Skip("needs root")
			}

			By("creating a new Docker session for testing")
			sess := Successful(morbyd.NewSession(ctx))
			DeferCleanup(func(ctx context.Context) {
				sess.Close(ctx)
			})

			By("spinning up a Docker container with CRI API providers, courtesy of the KinD k8s sig")
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
			//	 kindisch-...
			Expect(sess.BuildImage(ctx, "../../engineclient/cri/test/_kindisch",
				build.WithTag(img.Name),
				build.WithBuildArg("KINDEST_BASE_TAG="+test.KindestBaseImageTag),
				build.WithOutput(timestamper.New(GinkgoWriter)))).
				Error().NotTo(HaveOccurred())

			providerCntr = Successful(sess.Run(ctx, img.Name,
				run.WithName(kindischName),
				run.WithAutoRemove(),
				run.WithPrivileged(),
				run.WithSecurityOpt("label=disable"),
				run.WithCgroupnsMode("private"),
				run.WithVolume("/var"),
				run.WithVolume("/dev/mapper:/dev/mapper"),
				run.WithVolume("/lib/modules:/lib/modules:ro"),
				run.WithTmpfs("/tmp"),
				run.WithTmpfs("/run"),
				run.WithDevice("/dev/fuse"),
				run.WithCombinedOutput(timestamper.New(GinkgoWriter))))
			DeferCleanup(func(ctx context.Context) {
				By("removing the CRI API providers Docker container")
				providerCntr.Kill(ctx)
			})

			By("waiting for the CRI API provider to become responsive")
			pid := Successful(providerCntr.PID(ctx))
			// apipath must not include absolute symbolic links, but already be
			// properly resolved.
			endpoint := fmt.Sprintf("/proc/%d/root%s",
				pid, "/run/containerd/containerd.sock")
			var cricl *criengine.Client
			Eventually(func() error {
				var err error
				cricl, err = criengine.New(endpoint, criengine.WithTimeout(1*time.Second))
				return err
			}).Within(30*time.Second).ProbeEvery(1*time.Second).
				Should(Succeed(), "CRI API provider never became responsive")
			defer func() { cricl.Close() }()

		})

		var mw watcher.Watcher

		BeforeEach(func(ctx context.Context) {
			pid := Successful(providerCntr.PID(ctx))
			endpoint := fmt.Sprintf("/proc/%d/root%s",
				pid, "/run/containerd/containerd.sock")
			mw = Successful(New(endpoint, nil,
				criengine.WithPID(pid)))
			DeferCleanup(func() {
				mw.Close()
			})
			Expect(mw.PID()).To(Equal(pid))
		})

		It("gets and uses the underlying CRI client", slowSpec, func(ctx context.Context) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			cric, ok := mw.Client().(*criengine.Client)
			Expect(ok).To(BeTrue())
			Expect(cric).NotTo(BeNil())

			Expect(cric.RuntimeService().Version(ctx, &runtime.VersionRequest{})).
				Error().NotTo(HaveOccurred())
		})

		It("watches", slowSpec, func(ctx context.Context) {
			ctx, cancel := context.WithCancel(ctx)
			done := make(chan struct{})
			go func() {
				_ = mw.Watch(ctx)
				close(done)
			}()

			cric, ok := mw.Client().(*criengine.Client)
			Expect(ok).To(BeTrue())
			Expect(cric).NotTo(BeNil())

			By("checking that the watcher keeps watching")
			Consistently(done, "1s").ShouldNot(BeClosed())

			By("pulling the required canary image")
			Expect(cric.ImageService().PullImage(ctx, &runtime.PullImageRequest{
				Image: &runtime.ImageSpec{
					Image: "busybox:stable",
				},
			})).Error().NotTo(HaveOccurred())

			By("creating a new pod")
			podconfig := &runtime.PodSandboxConfig{
				Metadata: &runtime.PodSandboxMetadata{
					Name:      k8sTestPodName,
					Namespace: k8sTestNamespace,
					Uid:       uuid.NewString(),
				},
				Hostname: k8sTestPodName,
			}
			podr := Successful(cric.RuntimeService().RunPodSandbox(ctx, &runtime.RunPodSandboxRequest{
				Config: podconfig,
			}))
			DeferCleanup(func(ctx context.Context) {
				By("removing the pod")
				Expect(cric.RuntimeService().RemovePodSandbox(ctx, &runtime.RemovePodSandboxRequest{
					PodSandboxId: podr.PodSandboxId,
				})).Error().NotTo(HaveOccurred())
			})

			By("creating a container inside the pod")
			podcntr := Successful(cric.RuntimeService().CreateContainer(ctx, &runtime.CreateContainerRequest{
				PodSandboxId: podr.PodSandboxId,
				Config: &runtime.ContainerConfig{
					Metadata: &runtime.ContainerMetadata{
						Name: "hellorld",
					},
					Image: &runtime.ImageSpec{
						Image: "busybox:stable",
					},
					Command: []string{
						"/bin/sh",
						"-c",
						"mkdir -p /www && echo Hellorld!>/www/index.html && httpd -f -p 5099 -h /www",
					},
					Labels: map[string]string{
						"foo": "bar",
					},
					Annotations: map[string]string{
						"fools": "barz",
					},
				},
				SandboxConfig: podconfig,
			}))
			DeferCleanup(func() {
				By("removing the container")
				_, _ = cric.RuntimeService().RemoveContainer(ctx, &runtime.RemoveContainerRequest{
					ContainerId: podcntr.ContainerId,
				})
			})

			By("starting the container")
			Expect(cric.RuntimeService().StartContainer(ctx, &runtime.StartContainerRequest{
				ContainerId: podcntr.ContainerId,
			})).Error().NotTo(HaveOccurred())

			// eventually there should be a container poping up with the correct
			// namespace label.
			portfolio := func() []string {
				if proj := mw.Portfolio().Project(""); proj != nil {
					return proj.ContainerNames()
				}
				return []string{}
			}
			Eventually(portfolio).Within(5 * time.Second).ProbeEvery(250 * time.Millisecond).
				Should(ContainElement("hellorld"))

			// and eventually that container should also be gone from the watch list
			// after we killed it.
			By("removing the container")
			Expect(cric.RuntimeService().RemoveContainer(ctx, &runtime.RemoveContainerRequest{
				ContainerId: podcntr.ContainerId,
			})).Error().NotTo(HaveOccurred())
			Eventually(portfolio).Within(2 * time.Second).ProbeEvery(250 * time.Millisecond).
				Should(Not(ContainElement("hellorld")))

			// wait for the watcher to correctly spin down.
			cancel()
			Eventually(done).Should(BeClosed())
		})

	})

})
