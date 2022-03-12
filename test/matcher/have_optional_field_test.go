// Copyright 2022 Harald Albrecht.
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

package matcher

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("HaveOptionalField matcher", func() {

	type T struct {
		Foo string
	}
	st := T{Foo: "foo"}

	It("does not satisfy a missing field", func() {
		m := HaveOptionalField("Bar", "bar")
		success, err := m.Match(st)
		Expect(err).NotTo(HaveOccurred())
		Expect(success).To(BeFalse())
		Expect(m.FailureMessage(st)).To(MatchRegexp(`No field named 'Bar' in struct:`))
	})

	It("does not satisfy an existing field not satisfying the matcher", func() {
		m := HaveOptionalField("Foo", "bar")
		success, err := m.Match(st)
		Expect(err).NotTo(HaveOccurred())
		Expect(success).To(BeFalse())
		Expect(m.FailureMessage(st)).To(MatchRegexp(`Value for field 'Foo' failed to satisfy matcher.`))
		Expect(m.NegatedFailureMessage(st)).To(MatchRegexp(`Value for field 'Foo' satisfied matcher, but should not have.`))
	})

	It("matches a satisfying field (or not)", func() {
		Expect(st).To(HaveOptionalField("Foo", "foo"))
		Expect(st).NotTo(HaveOptionalField("Foo", "bar"))
	})

})
