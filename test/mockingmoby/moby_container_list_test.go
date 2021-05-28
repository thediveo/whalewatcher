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

package mockingmoby

import (
	"context"
	"errors"

	"github.com/docker/docker/api/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	. "github.com/thediveo/errxpect"
)

var _ = Describe("lists mocked containers", func() {

	It("lists containers", func() {
		mm := NewMockingMoby()
		defer mm.Close()

		cntrs, err := mm.ContainerList(context.Background(), types.ContainerListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cntrs).To(HaveLen(0))

		mm.AddContainer(mocking_moby)
		cntrs, err = mm.ContainerList(context.Background(), types.ContainerListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cntrs).To(HaveLen(1))
		c := cntrs[0]
		Expect(c.ID).To(Equal(mocking_moby.ID))
		Expect(c.Names).To(Equal([]string{"/" + mocking_moby.Name}))
		Expect(c.Labels).To(Equal(mocking_moby.Labels))
		Expect(c.Status).To(Equal(MockedStatus[mocking_moby.Status]))

		mm.AddContainer(furious_furuncle)
		cntrs, err = mm.ContainerList(context.Background(), types.ContainerListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(cntrs).To(HaveLen(2))
		Expect(cntrs).To(ConsistOf(
			MatchFields(IgnoreExtras, Fields{
				"ID": Equal(mocking_moby.ID),
			}),
			MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furious_furuncle.ID),
			}),
		))
	})

	It("recognizes cancelled context", func() {
		mm := NewMockingMoby()
		defer mm.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		Errxpect(mm.ContainerList(ctx, types.ContainerListOptions{})).To(HaveOccurred())
	})

	It("registers and calls hooks", func() {
		mm := NewMockingMoby()
		defer mm.Close()
		doh := errors.New("doh!")

		_, err := mm.ContainerInspect(
			WithHook(
				context.Background(),
				ContainerInspectPre,
				func(HookKey) error {
					return doh
				}), "foobar")
		Expect(err).To(Equal(doh))

		_, err = mm.ContainerInspect(
			WithHook(
				context.Background(),
				ContainerInspectPost,
				func(HookKey) error {
					return doh
				}), "foobar")
		Expect(err).To(Equal(doh))
	})

})
