// Copyright 2023 Harald Albrecht.
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
	"errors"

	"golang.org/x/exp/slices"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gcustom"
	"github.com/onsi/gomega/types"
)

// All succeeds only after it has been shown a sequence of actual values that
// “ticked off” the specified matchers in no particular order. All will return
// an error in case it encounters an actual value not matching any of the
// specified matchers. If the same match should occur multiple times it needs to
// be specified as many times as required.
func All(matchers ...types.GomegaMatcher) types.GomegaMatcher {
	// Now that's fun to keep state not in a struct "object" but instead capture
	// state in a closure.
	remaining := slices.Clone(matchers)
	return gcustom.MakeMatcher(func(actual any) (bool, error) {
		for idx, matcher := range remaining {
			succeeded, err := matcher.Match(actual)
			if err != nil {
				return false, err
			}
			if !succeeded {
				continue // ...maybe another one will match...?
			}
			remaining = slices.Delete(remaining, idx, idx+1)
			if len(remaining) > 0 {
				return false, nil
			}
			return true, nil
		}
		return false, errors.New(format.Message(actual, "not in expected set"))
	}).WithTemplate("Expected:\n{{.FormattedActual}}\n{{.To}} complete set")
}
