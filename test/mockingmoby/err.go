// Copyright 2026 Harald Albrecht.
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

import "fmt"

// OpaqueWrappingError wraps a containerd error with a custom error message,
// where the wrapped container error can be unwrapped, but unlike fmt.Errorf the
// wrapped error's Error message is never used.
type OpaqueWrappingError struct {
	msg     string
	wrapped error
}

func errwrap(err error, format string, args ...any) *OpaqueWrappingError {
	return &OpaqueWrappingError{
		msg:     fmt.Sprintf(format, args...),
		wrapped: err,
	}
}

func (e *OpaqueWrappingError) Error() string { return e.msg }

func (e *OpaqueWrappingError) Unwrap() error { return e.wrapped }
