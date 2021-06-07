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

package watcher

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("pause state queue", func() {

	It("never adds twice", func() {
		q := pendingPauseStates{}
		q.Add("foo", false)
		Expect(q).To(HaveLen(1))
		q.Add("foo", true)
		Expect(q).To(HaveLen(1))
		Expect(q[0]).To(Equal(pauseState{ID: "foo", Paused: true}))
	})

	It("removes", func() {
		q := pendingPauseStates{}
		q.Add("foo", false)
		q.Add("bar", true)
		Expect(q).To(HaveLen(2))
		q.Remove("foo")
		Expect(q).To(HaveLen(1))
		Expect(q[0]).To(Equal(pauseState{ID: "bar", Paused: true}))
	})

	It("keeps silent on removing pausing state for nonexisting ID", func() {
		q := pendingPauseStates{}
		Expect(func() { q.Remove("foo") }).NotTo(Panic())
	})

})
