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

package ctr

import (
	"strings"

	"github.com/ory/dockertest/v3"

	gi "github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
)

// Successfully runs a "ctr” command inside the specified Docker container that
// not only contains a containerd but also a ctr binary, expecting the command
// to succeed without any error and a zero exit code.
func Successfully(cntr *dockertest.Resource, args ...string) {
	gi.GinkgoHelper()
	exitCode, err := cntr.Exec(
		append([]string{"ctr"}, args...),
		dockertest.ExecOptions{
			// https://github.com/ory/dockertest/issues/472

			//StdOut: gi.GinkgoWriter,
			//StdErr: gi.GinkgoWriter,
			TTY: false,
		},
	)
	g.Expect(err).NotTo(g.HaveOccurred(), "failed: ctr %s", strings.Join(args, " "))
	g.Expect(exitCode).To(g.BeZero(), "failed with non-zero exit code: ctr %s", strings.Join(args, " "))
}

// Exec runs a "ctr" command inside the specified Docker container that not only
// contains a containerd but also a ctr binary, returning ctr's exit code.
func Exec(cntr *dockertest.Resource, args ...string) int {
	gi.GinkgoHelper()
	exitCode, err := cntr.Exec(
		append([]string{"ctr"}, args...),
		dockertest.ExecOptions{
			//StdOut: gi.GinkgoWriter,
			//StdErr: gi.GinkgoWriter,
			TTY: false,
		},
	)
	g.Expect(err).NotTo(g.HaveOccurred(), "failed: ctr %s", strings.Join(args, " "))
	return exitCode
}
