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
	. "github.com/onsi/ginkgo/v2"
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

		mm.AddContainer(mockingMoby)
		Consistently(evs).ShouldNot(Receive())
		Consistently(errs).ShouldNot(Receive())

		mm.AddContainer(furiousFuruncle)
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(events.ContainerEventType),
			"Action": Equal("start"),
			"Scope":  Equal("local"),
			"Actor": MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furiousFuruncle.ID),
				"Attributes": And(
					HaveKeyWithValue("name", furiousFuruncle.Name),
					HaveKeyWithValue("foo", "bar"),
				),
			}),
		})))
		Consistently(errs).ShouldNot(Receive())

		mm.PauseContainer(furiousFuruncle.Name)
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(events.ContainerEventType),
			"Action": Equal("pause"),
			"Scope":  Equal("local"),
			"Actor": MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furiousFuruncle.ID),
				"Attributes": And(
					HaveKeyWithValue("name", furiousFuruncle.Name),
					HaveKeyWithValue("foo", "bar"),
				),
			}),
		})))
		mm.PauseContainer(furiousFuruncle.Name)
		Consistently(evs).ShouldNot(Receive())

		mm.UnpauseContainer(furiousFuruncle.Name)
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(events.ContainerEventType),
			"Action": Equal("unpause"),
			"Scope":  Equal("local"),
			"Actor": MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furiousFuruncle.ID),
				"Attributes": And(
					HaveKeyWithValue("name", furiousFuruncle.Name),
					HaveKeyWithValue("foo", "bar"),
				),
			}),
		})))
		mm.UnpauseContainer(furiousFuruncle.Name)
		Consistently(evs).ShouldNot(Receive())

		mm.RemoveContainer(furiousFuruncle.ID)
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Type":   Equal(events.ContainerEventType),
			"Action": Equal("die"),
			"Scope":  Equal("local"),
			"Actor": MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furiousFuruncle.ID),
				"Attributes": And(
					HaveKeyWithValue("name", furiousFuruncle.Name),
					HaveKeyWithValue("foo", "bar"),
				),
			}),
		})))
		Consistently(errs).ShouldNot(Receive())

		mm.AddContainer(furiousFuruncle)
		Eventually(evs).Should(Receive())
		Consistently(errs).ShouldNot(Receive())
		mm.StopContainer(furiousFuruncle.ID)
		Eventually(evs).Should(Receive(MatchFields(IgnoreExtras, Fields{
			"Action": Equal("die"),
			"Actor": MatchFields(IgnoreExtras, Fields{
				"ID": Equal(furiousFuruncle.ID),
			}),
		})))
		Consistently(errs).ShouldNot(Receive())
		mm.StopContainer(furiousFuruncle.ID)
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

		mm.AddContainer(furiousFuruncle)
		Consistently(evs).ShouldNot(Receive())
	})

})
