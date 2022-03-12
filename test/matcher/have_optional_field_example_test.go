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
	"github.com/thediveo/whalewatcher"

	. "github.com/onsi/gomega"
)

func ExampleHaveOptionalField() {
	project := whalewatcher.ComposerProject{
		Name: "myproj",
	}
	// Do not pass ComposerProject by value, as it contains a lock.
	Expect(&project).To(HaveOptionalField("Name", "myproj"))
	Expect(&project).NotTo(HaveOptionalField("Name", "steven's-proj"))
	Expect(&project).NotTo(HaveOptionalField("ID", "1234"))
	// Output:
}
