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

	"github.com/docker/docker/api/types/container"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

var _ = Describe("lists mocked containers", func() {

	It("lists containers", func() {
		mm := NewMockingMoby()
		defer mm.Close()

		cntrs := Successful(mm.ContainerList(context.Background(), container.ListOptions{}))
		Expect(cntrs).To(HaveLen(0))

		mm.AddContainer(mockingMoby)
		cntrs = Successful(mm.ContainerList(context.Background(), container.ListOptions{}))
		Expect(cntrs).To(HaveLen(1))
		c := cntrs[0]
		Expect(c.ID).To(Equal(mockingMoby.ID))
		Expect(c.Names).To(Equal([]string{"/" + mockingMoby.Name}))
		Expect(c.Labels).To(Equal(mockingMoby.Labels))
		Expect(c.Status).To(Equal(MockedStatus[mockingMoby.Status]))

		mm.AddContainer(furiousFuruncle)
		cntrs = Successful(mm.ContainerList(context.Background(), container.ListOptions{}))
		Expect(cntrs).To(HaveLen(2))
		Expect(cntrs).To(ConsistOf(
			HaveField("ID", Equal(mockingMoby.ID)),
			HaveField("ID", Equal(furiousFuruncle.ID)),
		))
	})

	It("recognizes cancelled context", func() {
		mm := NewMockingMoby()
		defer mm.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		Expect(mm.ContainerList(ctx, container.ListOptions{})).Error().To(HaveOccurred())
	})

	It("registers and calls hooks", func() {
		mm := NewMockingMoby()
		defer mm.Close()
		doh := errors.New("doh!")

		Expect(mm.ContainerInspect(
			WithHook(
				context.Background(),
				ContainerInspectPre,
				func(HookKey) error {
					return doh
				}), "foobar")).
			Error().To(Equal(doh))

		Expect(mm.ContainerInspect(
			WithHook(
				context.Background(),
				ContainerInspectPost,
				func(HookKey) error {
					return doh
				}), "foobar")).
			Error().To(Equal(doh))
	})

})
