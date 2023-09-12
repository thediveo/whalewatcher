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
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/thediveo/once"
	"github.com/thediveo/whalewatcher"
	"github.com/thediveo/whalewatcher/engineclient"
	"github.com/thediveo/whalewatcher/test/matcher"
	rtv1 "k8s.io/cri-api/pkg/apis/runtime/v1"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	. "github.com/thediveo/success"
)

const kindestBaseTag = "v20230525-4c49613f"

const kindischName = "kindisch-ww-cri"
const testNamespace = "wwcritest"
const testPod = "wwcritestpod"

var _ = Describe("CRI API", Ordered, func() {

	const withSandboxEvents = true

	var providerCntr *dockertest.Resource

	BeforeAll(func(ctx context.Context) {
		if os.Getuid() != 0 {
			Skip("needs root")
		}

		By("spinning up a Docker container with CRI API providers, courtesy of the KinD k8s sig")
		pool := Successful(dockertest.NewPool("unix:///var/run/docker.sock"))
		_ = pool.RemoveContainerByName(kindischName)
		// The necessary container start arguments come from KinD's Docker node
		// provisioner, see:
		// https://github.com/kubernetes-sigs/kind/blob/3610f606516ccaa88aa098465d8c13af70937050/pkg/cluster/internal/providers/docker/provision.go#L133
		//
		// Please note that --privileged already implies switching off AppArmor
		//
		// docker run -it --rm --name kindisch
		//   --privileged
		//   --cgroupns=private
		//   --init=false
		//   --volume /dev/mapper:/dev/mapper
		//   --device /dev/fuse
		//   --tmpfs /tmp
		//   --tmpfs /run
		//   --volume /var
		//   --volume /lib/modules:/lib/modules:ro ww-containerd-in-docker-test
		providerCntr = Successful(pool.BuildAndRunWithBuildOptions(
			&dockertest.BuildOptions{
				ContextDir: "./test/kindisch", // sorry, couldn't resist the pun.
				Dockerfile: "Dockerfile",
				BuildArgs: []docker.BuildArg{
					{Name: "KINDEST_BASE_TAG", Value: kindestBaseTag},
				},
			},
			&dockertest.RunOptions{
				Name:       kindischName,
				Privileged: true,
				Mounts: []string{
					"/dev/mapper:/dev/mapper",
					"/var",
					"/lib/modules:/lib/modules:ro",
				},
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
			By("removing the CRI API providers Docker container")
			Expect(pool.Purge(providerCntr)).To(Succeed())
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
			Expect(providerCntr.Container.State.Pid).NotTo(BeZero())
			// apipath must not include absolute symbolic links, but already be
			// properly resolved.
			endpoint := fmt.Sprintf("/proc/%d/root%s",
				providerCntr.Container.State.Pid, apipath)
			Eventually(func() error {
				var err error
				cricl, err = New(endpoint)
				return err
			}).Within(30*time.Second).ProbeEvery(100*time.Millisecond).
				Should(Succeed(), "CRI API provider never became responsive")
			DeferCleanup(func() {
				cricl.Close()
				cricl = nil
			})

			By("waiting for the CRI API to become fully operational", func() {
				Eventually(ctx, func(ctx context.Context) error {
					_, err := cricl.rtcl.Status(ctx, &rtv1.StatusRequest{})
					return err
				}).ProbeEvery(250 * time.Millisecond).
					Should(Succeed())
			})

			By("pulling the required canary image")
			Expect(cricl.imgcl.PullImage(ctx, &rtv1.PullImageRequest{
				Image: &rtv1.ImageSpec{
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
			Expect(cricl.rtcl.Version(ctx, &v1.VersionRequest{})).To(SatisfyAll(
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
			podconfig := &rtv1.PodSandboxConfig{
				Metadata: &rtv1.PodSandboxMetadata{
					Name:      testPod,
					Namespace: testNamespace,
					Uid:       uuid.NewString(),
				},
			}
			podsbox := Successful(cricl.rtcl.RunPodSandbox(ctx, &rtv1.RunPodSandboxRequest{
				Config: podconfig,
			}))
			defer func() {
				By("removing the pod")
				Expect(cricl.rtcl.RemovePodSandbox(ctx, &rtv1.RemovePodSandboxRequest{
					PodSandboxId: podsbox.PodSandboxId,
				})).Error().NotTo(HaveOccurred())
			}()

			By("creating a container inside the pod")
			podcntr := Successful(cricl.rtcl.CreateContainer(ctx, &rtv1.CreateContainerRequest{
				PodSandboxId: podsbox.PodSandboxId,
				Config: &rtv1.ContainerConfig{
					Metadata: &rtv1.ContainerMetadata{
						Name: "hellorld",
					},
					Image: &rtv1.ImageSpec{
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
				Expect(cricl.rtcl.RemoveContainer(ctx, &rtv1.RemoveContainerRequest{
					ContainerId: podcntr.ContainerId,
				})).Error().NotTo(HaveOccurred())
			}()

			By("starting the container")
			Expect(cricl.rtcl.StartContainer(ctx, &rtv1.StartContainerRequest{
				ContainerId: podcntr.ContainerId,
			})).Error().NotTo(HaveOccurred())

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

			cntrs := Successful(cw.List(ctx))
			Expect(cntrs).To(ContainElement(And(
				HaveField("Name", "hellorld"),
				HaveField("PID", cntr.PID),
			)))
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
			podconfig := &rtv1.PodSandboxConfig{
				Metadata: &rtv1.PodSandboxMetadata{
					Name:      testPod,
					Namespace: testNamespace,
					Uid:       uuid.NewString(),
				},
				Hostname: testPod,
			}
			podr := Successful(cricl.rtcl.RunPodSandbox(ctx, &rtv1.RunPodSandboxRequest{
				Config: podconfig,
			}))
			DeferCleanup(func(ctx context.Context) {
				By("cleaning up: removing the pod")
				Expect(cricl.rtcl.RemovePodSandbox(ctx, &rtv1.RemovePodSandboxRequest{
					PodSandboxId: podr.PodSandboxId,
				})).Error().NotTo(HaveOccurred())
			})

			if withSandboxEvents {
				By("waiting for the sandbox started event")
				Eventually(cntrevch).Within(5 * time.Second).ProbeEvery(100 * time.Millisecond).
					Should(Receive(And(
						HaveField("Type", engineclient.ContainerStarted),
						HaveField("ID", podr.PodSandboxId),
					)))
			}

			By("creating a container inside the pod")
			podcntr := Successful(cricl.rtcl.CreateContainer(ctx, &rtv1.CreateContainerRequest{
				PodSandboxId: podr.PodSandboxId,
				Config: &rtv1.ContainerConfig{
					Metadata: &rtv1.ContainerMetadata{
						Name: "hellorld",
					},
					Image: &rtv1.ImageSpec{
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
				Expect(cricl.rtcl.RemoveContainer(ctx, &rtv1.RemoveContainerRequest{
					ContainerId: podcntr.ContainerId,
				})).Error().To(Or(
					Not(HaveOccurred()),
					MatchError(ContainSubstring("code = NotFound"))))
			})

			By("starting the container")
			Expect(cricl.rtcl.StartContainer(ctx, &rtv1.StartContainerRequest{
				ContainerId: podcntr.ContainerId,
			})).Error().NotTo(HaveOccurred())

			By("waiting for the container started event")
			Eventually(cntrevch).Within(5 * time.Second).ProbeEvery(100 * time.Millisecond).
				Should(Receive(And(
					HaveField("Type", engineclient.ContainerStarted),
					HaveField("ID", podcntr.ContainerId),
				)))

			By("removing the pod")
			Expect(cricl.rtcl.RemovePodSandbox(ctx, &rtv1.RemovePodSandboxRequest{
				PodSandboxId: podr.PodSandboxId,
			})).Error().NotTo(HaveOccurred())

			By("waiting for the container and pod stopped events")
			expected := []types.GomegaMatcher{
				And(
					HaveField("Type", engineclient.ContainerExited),
					HaveField("ID", podcntr.ContainerId),
				),
			}
			if withSandboxEvents {
				expected = append(expected, And(
					HaveField("Type", engineclient.ContainerExited),
					HaveField("ID", podr.PodSandboxId),
				))
			}
			Eventually(cntrevch).Within(5 * time.Second).ProbeEvery(100 * time.Millisecond).
				Should(Receive(matcher.All(expected...)))

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
