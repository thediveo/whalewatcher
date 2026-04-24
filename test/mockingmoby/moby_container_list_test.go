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

	"github.com/moby/moby/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var _ = Describe("lists mocked containers", func() {

	It("lists containers", func() {
		mm := NewMockingMoby()
		defer func() { _ = mm.Close() }()

		cntrs := Successful(mm.ContainerList(context.Background(), client.ContainerListOptions{}))
		Expect(cntrs.Items).To(BeEmpty())

		mm.AddContainer(mockingMoby)
		cntrs = Successful(mm.ContainerList(context.Background(), client.ContainerListOptions{}))
		Expect(cntrs.Items).To(HaveLen(1))
		c := cntrs.Items[0]
		Expect(c.ID).To(Equal(mockingMoby.ID))
		Expect(c.Names).To(Equal([]string{"/" + mockingMoby.Name}))
		Expect(c.Labels).To(Equal(mockingMoby.Labels))
		Expect(c.Status).To(BeEmpty())
		Expect(c.State).To(Equal(MockedContainerStates[mockingMoby.Status]))

		mm.AddContainer(furiousFuruncle)
		cntrs = Successful(mm.ContainerList(context.Background(), client.ContainerListOptions{}))
		Expect(cntrs.Items).To(HaveLen(2))
		Expect(cntrs.Items).To(ConsistOf(
			HaveField("ID", Equal(mockingMoby.ID)),
			HaveField("ID", Equal(furiousFuruncle.ID)),
		))
	})

	It("recognizes cancelled context", func() {
		mm := NewMockingMoby()
		defer func() { _ = mm.Close() }()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		Expect(mm.ContainerList(ctx, client.ContainerListOptions{})).Error().To(HaveOccurred())
	})

	It("registers and calls hooks", func() {
		mm := NewMockingMoby()
		defer func() { _ = mm.Close() }()
		doh := errors.New("doh!")

		Expect(mm.ContainerInspect(
			WithHook(
				context.Background(),
				ContainerInspectPre,
				func(HookKey) error {
					return doh
				}), "foobar", client.ContainerInspectOptions{})).
			Error().To(BeEquivalentTo(doh))

		Expect(mm.ContainerInspect(
			WithHook(
				context.Background(),
				ContainerInspectPost,
				func(HookKey) error {
					return doh
				}), "foobar", client.ContainerInspectOptions{})).
			Error().To(BeEquivalentTo(doh))
	})

})
