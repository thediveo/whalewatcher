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

var _ = Describe("inspects mocked containers", func() {

	It("inspects containers by ID and name", func() {
		mm := NewMockingMoby()
		defer mm.Close()

		Errxpect(mm.ContainerInspect(context.Background(), "foo")).To(HaveOccurred())

		mm.AddContainer(furious_furuncle)
		details, err := mm.ContainerInspect(context.Background(), furious_furuncle.ID)
		Expect(err).NotTo(HaveOccurred())
		cmatcher := MatchFields(IgnoreExtras, Fields{
			"ContainerJSONBase": PointTo(MatchFields(IgnoreExtras, Fields{
				"ID":   Equal(furious_furuncle.ID),
				"Name": Equal("/" + furious_furuncle.Name),
				"State": PointTo(MatchFields(IgnoreExtras, Fields{
					"Status":  Equal(MockedStatus[furious_furuncle.Status]),
					"Running": BeTrue(),
					"Paused":  BeFalse(),
					"Pid":     Equal(furious_furuncle.PID),
				})),
			})),
			"Config": PointTo(MatchFields(IgnoreExtras, Fields{
				"Labels": Equal(furious_furuncle.Labels),
			})),
		})
		Expect(details).To(cmatcher)

		details, err = mm.ContainerInspect(context.Background(), furious_furuncle.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(details).To(cmatcher)
	})

	It("inspects status correctly", func() {
		mm := NewMockingMoby()
		defer mm.Close()
		mm.AddContainer(furious_furuncle)
		mm.StopContainer(furious_furuncle.Name)
		details, err := mm.ContainerInspect(context.Background(), furious_furuncle.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(details).To(MatchFields(IgnoreExtras, Fields{
			"ContainerJSONBase": PointTo(MatchFields(IgnoreExtras, Fields{
				"ID":   Equal(furious_furuncle.ID),
				"Name": Equal("/" + furious_furuncle.Name),
				"State": PointTo(MatchFields(IgnoreExtras, Fields{
					"Status":  Equal(MockedStatus[MockedExited]),
					"Running": BeFalse(),
					"Paused":  BeFalse(),
					"Pid":     BeZero(),
				})),
			})),
			"Config": PointTo(MatchFields(IgnoreExtras, Fields{
				"Labels": Equal(furious_furuncle.Labels),
			})),
		}))

		mm.AddContainer(pausing_pm)
		details, err = mm.ContainerInspect(context.Background(), pausing_pm.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(details).To(MatchFields(IgnoreExtras, Fields{
			"ContainerJSONBase": PointTo(MatchFields(IgnoreExtras, Fields{
				"ID": Equal(pausing_pm.ID),
				"State": PointTo(MatchFields(IgnoreExtras, Fields{
					"Status":  Equal(MockedStatus[pausing_pm.Status]),
					"Running": BeTrue(),
					"Paused":  BeTrue(),
				})),
			})),
		}))
	})

	It("recognizes cancelled context", func() {
		mm := NewMockingMoby()
		defer mm.Close()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		Errxpect(mm.ContainerInspect(ctx, "foo")).To(HaveOccurred())
	})

	It("registers and calls hooks", func() {
		mm := NewMockingMoby()
		defer mm.Close()
		doh := errors.New("doh!")

		cntrs, err := mm.ContainerList(
			WithHook(
				context.Background(),
				ContainerListPost,
				func(key HookKey) error {
					Expect(key).To(Equal(ContainerListPost))
					return doh
				}), types.ContainerListOptions{})
		Expect(err).To(Equal(doh))
		Expect(cntrs).To(BeNil())

		_, err = mm.ContainerList(
			WithHook(
				context.Background(),
				ContainerListPre,
				func(HookKey) error {
					return doh
				}), types.ContainerListOptions{})
		Expect(err).To(Equal(doh))
	})

})
