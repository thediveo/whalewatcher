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

var _ = Describe("HaveName matcher", func() {

	It("matches", func() {
		type T struct {
			Name string
		}
		type K struct {
			Foo string
		}

		t := T{Name: "FOO"}
		Expect(t).To(HaveName(t.Name))
		Expect(t).NotTo(HaveName("BAR"))

		k := K{Foo: "FOO"}
		Expect(HaveName(k.Foo).Match(k)).Error().To(HaveOccurred())
	})

})