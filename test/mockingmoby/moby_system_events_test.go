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

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("mocked event streaming", func() {

	It("streams container events", func() {
		mm := NewMockingMoby()
		defer mm.Close()

		ctx, cancel := context.WithCancel(context.Background())

		evs, errs := mm.Events(ctx, types.EventsOptions{})
		Expect(evs).NotTo(BeNil())
		Expect(errs).NotTo(BeNil())

		mm.AddContainer(mocking_moby)
		Consistently(evs).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())

		mm.AddContainer(furious_furuncle)
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(events.ContainerEventType),
			"Action": Equal("start"),
			"Scope":  Equal("local"),
			"Actor": MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furious_furuncle.ID),
				"Attributes": And(
					HaveKeyWithValue("name", furious_furuncle.Name),
					HaveKeyWithValue("foo", "bar"),
				),
			}),
		})))
		Consistently(errs).ShouldNot(Receive())

		mm.RemoveContainer(furious_furuncle.ID)
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(events.ContainerEventType),
			"Action": Equal("die"),
			"Scope":  Equal("local"),
			"Actor": MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furious_furuncle.ID),
				"Attributes": And(
					HaveKeyWithValue("name", furious_furuncle.Name),
					HaveKeyWithValue("foo", "bar"),
				),
			}),
		})))
		Consistently(errs).ShouldNot(Receive())

		mm.AddContainer(furious_furuncle)
		Eventually(evs).Should(Receive())
		Consistently(errs).ShouldNot(Receive())
		mm.StopContainer(furious_furuncle.ID)
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Action": Equal("die"),
			"Actor": MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furious_furuncle.ID),
			}),
		})))
		Consistently(errs).ShouldNot(Receive())
		mm.StopContainer(furious_furuncle.ID)
		Consistently(evs).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())

		cancel()
		Eventually(errs).Should(Receive(Equal(ctx.Err())))
	})

	It("stops event streaming", func() {
		mm := NewMockingMoby()
		defer mm.Close()

		evs, errs := mm.Events(context.Background(), types.EventsOptions{})
		Expect(evs).NotTo(BeNil())
		Expect(errs).NotTo(BeNil())

		mm.StopEvents()
		Eventually(errs).Should(Receive(Equal(ErrEventStreamStopped)))

		mm.AddContainer(furious_furuncle)
		Consistently(evs).ShouldNot(Receive())
	})

})
