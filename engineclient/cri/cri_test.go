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
	"github.com/thediveo/morbyd/session"
	"github.com/thediveo/morbyd/timestamper"
	"github.com/thediveo/once"
	"github.com/thediveo/whalewatcher"
	"github.com/thediveo/whalewatcher/engineclient"
	"github.com/thediveo/whalewatcher/engineclient/cri/test/img"
	"github.com/thediveo/whalewatcher/test"
	"github.com/thediveo/whalewatcher/test/matcher"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
	. "github.com/thediveo/whalewatcher/test/matcher"
)

const (
	// // name of Docker container with containerd and cri-o engines
	kindischName = "ww-engineclient-cri"

	k8sTestNamespace = "wwcritest"
	k8sTestPodName   = "wwcritestpod"
)

// Please note that these tests assume that pod sandboxes also get reported
// through events.
var _ = Describe("CRI API engineclient", Ordered, func() {

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
		sess := Successful(morbyd.NewSession(ctx,
			session.WithAutoCleaning("test.whalewatcher=engineclient/cri")))
		DeferCleanup(func(ctx context.Context) {
			By("auto-cleaning the session")
			sess.Close(ctx)
		})

		By("spinning up a Docker container with CRI API providers, courtesy of the KinD k8s sig")
		// The necessary container start arguments come from KinD's Docker node
		// provisioner, see:
		// https://github.com/kubernetes-sigs/kind/blob/3610f606516ccaa88aa098465d8c13af70937050/pkg/cluster/internal/providers/docker/provision.go#L133
		//
		// Please note that --privileged already implies switching off AppArmor.
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
		Expect(sess.BuildImage(ctx, "./test/_kindisch",
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
	})

	// In the following, we want to run the set of unit tests on multiple CRI
	// API providers. This is kind of what normally Gingko's DescribeTable is
	// for, but unfortunately we need kind of BeforeEach (table entry) and this
	// isn't available in Ginkgo. So we need to do it manually.

	var cricl *Client
	var cw *CRIWatcher

	beforeEachWithAPIPath := func(apipath string) func(context.Context) {
		return func(ctx context.Context) {
			By("waiting for the CRI API provider to become responsive")
			pid := Successful(providerCntr.PID(ctx))
			// apipath must not include absolute symbolic links, but already be
			// properly resolved.
			endpoint := fmt.Sprintf("/proc/%d/root%s", pid, apipath)
			Eventually(func() error {
				var err error
				cricl, err = New(endpoint, WithTimeout(1*time.Second))
				return err
			}).Within(30*time.Second).ProbeEvery(1*time.Second).
				Should(Succeed(), "CRI API provider never became responsive")
			DeferCleanup(func() {
				cricl.Close()
				cricl = nil
			})
			Expect(cricl.Address()).To(HaveSuffix(endpoint))

			By("waiting for the CRI API to become fully operational", func() {
				Eventually(ctx, func(ctx context.Context) error {
					_, err := cricl.rtcl.Status(ctx, &runtime.StatusRequest{})
					return err
				}).ProbeEvery(250 * time.Millisecond).
					Should(Succeed())
			})

			By("pulling the required canary image")
			Expect(cricl.imgcl.PullImage(ctx, &runtime.PullImageRequest{
				Image: &runtime.ImageSpec{
					Image: "busybox:stable",
				},
			})).Error().NotTo(HaveOccurred())

			By("creating a CRI watcher")
			cw = NewCRIWatcher(cricl)
			DeferCleanup(func() {
				cw.Close()
				cw = nil
			})

			By("fetching API information")
			Expect(cricl.rtcl.Version(ctx, &runtime.VersionRequest{})).To(SatisfyAll(
				HaveField("Version", Not(BeEmpty())),
				HaveField("RuntimeName", Not(BeEmpty())),
			))
		}
	}

	tests := func() {

		It("inspects nil when container doesn't exist", func(ctx context.Context) {
			Expect(cw.Inspect(ctx, "---noid---")).Error().To(HaveOccurred())
		})

		It("lists and inspects an existing container", func(ctx context.Context) {
			By("creating a new pod")
			podconfig := &runtime.PodSandboxConfig{
				Metadata: &runtime.PodSandboxMetadata{
					Name:      k8sTestPodName,
					Namespace: k8sTestNamespace,
					Uid:       uuid.NewString(),
				},
			}
			podsbox := Successful(cricl.rtcl.RunPodSandbox(ctx, &runtime.RunPodSandboxRequest{
				Config: podconfig,
			}))
			defer func() {
				By("removing the pod")
				Expect(cricl.rtcl.RemovePodSandbox(ctx, &runtime.RemovePodSandboxRequest{
					PodSandboxId: podsbox.PodSandboxId,
				})).Error().NotTo(HaveOccurred())
			}()

			By("creating a container inside the pod")
			podcntr := Successful(cricl.rtcl.CreateContainer(ctx, &runtime.CreateContainerRequest{
				PodSandboxId: podsbox.PodSandboxId,
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
			defer func() {
				By("removing the container")
				Expect(cricl.rtcl.RemoveContainer(ctx, &runtime.RemoveContainerRequest{
					ContainerId: podcntr.ContainerId,
				})).Error().NotTo(HaveOccurred())
			}()

			By("starting the container")
			Expect(cricl.rtcl.StartContainer(ctx, &runtime.StartContainerRequest{
				ContainerId: podcntr.ContainerId,
			})).Error().NotTo(HaveOccurred())

			By("inspecting the container")
			var cntr *whalewatcher.Container
			Eventually(ctx, func() *whalewatcher.Container {
				cntr = Successful(cw.Inspect(ctx, podcntr.ContainerId))
				return cntr
			}).Within(5 * time.Second).ProbeEvery(250 * time.Millisecond).Should(Not(BeNil()))
			Expect(cntr).NotTo(BeNil())
			Expect(cntr.Name).To(Equal("hellorld"))
			Expect(cntr.ID).To(Equal(podcntr.ContainerId))
			Expect(cntr.PID).NotTo(BeZero())
			Expect(cntr.Labels).To(HaveKeyWithValue("foo", "bar"))
			Expect(cntr.Labels).To(HaveKeyWithValue(AnnotationKeyPrefix+"fools", "barz"))

			By("listing the container and the sandbox")
			cntrs := Successful(cw.List(ctx))
			Expect(cntrs).To(ConsistOf(
				And(
					HaveField("Name", "hellorld"),
					HaveField("PID", cntr.PID),
					HaveField("Labels", And(
						HaveKeyWithValue(PodNamespaceLabel, k8sTestNamespace),
						HaveKeyWithValue(PodNameLabel, k8sTestPodName),
						Not(HaveKey(PodSandboxLabel)),
					)),
				),
				And(
					HaveField("Labels", And(
						HaveKeyWithValue(PodNamespaceLabel, k8sTestNamespace),
						HaveKeyWithValue(PodNameLabel, k8sTestPodName),
						HaveKeyWithValue(PodSandboxLabel, ""),
					)),
				),
			))
		})

		It("watches", func(ctx context.Context) {
			ctx, cancel := context.WithCancel(ctx)
			closeOnce := once.Once(func() { cancel() }).Do
			defer closeOnce()

			cntrevch, errch := cw.LifecycleEvents(ctx)

			By("ensuring there is no event error")
			select {
			case err := <-errch:
				Expect(err).NotTo(HaveOccurred())
				panic("fallen off the disc world")
			case <-time.After(2 * time.Second):
			}

			By("creating a new pod")
			podconfig := &runtime.PodSandboxConfig{
				Metadata: &runtime.PodSandboxMetadata{
					Name:      k8sTestPodName,
					Namespace: k8sTestNamespace,
					Uid:       uuid.NewString(),
				},
				Hostname: k8sTestPodName,
			}
			podr := Successful(cricl.rtcl.RunPodSandbox(ctx, &runtime.RunPodSandboxRequest{
				Config: podconfig,
			}))
			DeferCleanup(func(ctx context.Context) {
				By("cleaning up: removing the pod")
				Expect(cricl.rtcl.RemovePodSandbox(ctx, &runtime.RemovePodSandboxRequest{
					PodSandboxId: podr.PodSandboxId,
				})).Error().NotTo(HaveOccurred())
			})

			By("waiting for the sandbox started event")
			Eventually(cntrevch).Within(5 * time.Second).ProbeEvery(100 * time.Millisecond).
				Should(Receive(And(
					HaveTimestamp(Not(BeZero())),
					HaveField("Type", engineclient.ContainerStarted),
					HaveField("ID", podr.PodSandboxId),
				)))

			By("creating a container inside the pod")
			podcntr := Successful(cricl.rtcl.CreateContainer(ctx, &runtime.CreateContainerRequest{
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
			DeferCleanup(func(ctx context.Context) {
				By("cleaning up: removing the container")
				Expect(cricl.rtcl.RemoveContainer(ctx, &runtime.RemoveContainerRequest{
					ContainerId: podcntr.ContainerId,
				})).Error().To(Or(
					Not(HaveOccurred()),
					MatchError(ContainSubstring("code = NotFound"))))
			})

			By("starting the container")
			Expect(cricl.rtcl.StartContainer(ctx, &runtime.StartContainerRequest{
				ContainerId: podcntr.ContainerId,
			})).Error().NotTo(HaveOccurred())

			By("waiting for the container started event")
			Eventually(cntrevch).Within(5 * time.Second).ProbeEvery(100 * time.Millisecond).
				Should(Receive(And(
					HaveField("Type", engineclient.ContainerStarted),
					HaveField("ID", podcntr.ContainerId),
				)))

			By("removing the pod")
			Expect(cricl.rtcl.RemovePodSandbox(ctx, &runtime.RemovePodSandboxRequest{
				PodSandboxId: podr.PodSandboxId,
			})).Error().NotTo(HaveOccurred())

			By("waiting for the container and pod stopped events")
			Eventually(cntrevch).Within(5 * time.Second).ProbeEvery(100 * time.Millisecond).
				Should(Receive(matcher.All(
					And(
						HaveTimestamp(Not(BeZero())),
						HaveField("Type", engineclient.ContainerExited),
						HaveField("ID", podcntr.ContainerId),
					),
					And(
						HaveTimestamp(Not(BeZero())),
						HaveField("Type", engineclient.ContainerExited),
						HaveField("ID", podr.PodSandboxId),
					),
				)))

			closeOnce()
			Eventually(errch).Should(BeClosed())
		})

	}

	When("using containerd", func() {
		BeforeAll(beforeEachWithAPIPath("/run/containerd/containerd.sock"))
		tests()
	})

	When("using cri-o", func() {
		BeforeAll(beforeEachWithAPIPath("/run/crio/crio.sock"))
		tests()
	})

})
