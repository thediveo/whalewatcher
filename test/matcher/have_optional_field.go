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
	"fmt"
	"regexp"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/matchers"
	"github.com/onsi/gomega/types"
)

// HaveOptionalField succeeds if actual is a struct and the value of the
// specified field matches the specified matcher; it is no error for the field
// to be non-existing, this matcher then does not fail, but it also does not
// succeed either.
func HaveOptionalField(field string, expected interface{}) types.GomegaMatcher {
	return &haveOptionalFieldMatcher{
		fm: matchers.HaveFieldMatcher{
			Field:    field,
			Expected: expected,
		},
	}
}

// haveOptionalFieldMatcher implements an optional field matcher by embedding
// Gomega's stock have-field matcher and wrapping its Match method in order to
// catch and gracefully handle missing field errors.
type haveOptionalFieldMatcher struct {
	hasField bool
	fm       matchers.HaveFieldMatcher
}

// Match almost works like Gomega's HaveFieldMatcher.Match, but ignores any
// missing field errors and instead of returning an error it just does not
// succeed.
func (matcher *haveOptionalFieldMatcher) Match(actual interface{}) (success bool, err error) {
	success, err = matcher.fm.Match(actual)
	if err != nil && reFieldError.MatchString(err.Error()) {
		return false, nil
	}
	matcher.hasField = true
	return
}

var reFieldError = regexp.MustCompile(`HaveField could not find (field|method) named '.*' in struct`)

// FailureMessage describes why this matcher failed to match when it was
// expected to do so.
func (matcher *haveOptionalFieldMatcher) FailureMessage(actual interface{}) (message string) {
	if !matcher.hasField {
		return fmt.Sprintf("No field named '%s' in struct:\n%s", matcher.fm.Field, format.Object(actual, 1))
	}
	return matcher.fm.FailureMessage(actual)
}

// NegatedFailureMessage describes why this matcher matched when it was not
// expected to do so.
func (matcher *haveOptionalFieldMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return matcher.fm.NegatedFailureMessage(actual)
}
