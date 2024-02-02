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

package cri

import (
	"context"
	"os"
	"syscall"

	"github.com/thediveo/morbyd"
	"github.com/thediveo/morbyd/run"
	"github.com/thediveo/morbyd/session"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
)

const (
	testName     = "ww-cri-uts"
	testHostname = "ohwwcrickety"
)

var _ = Describe("hostname", Ordered, func() {

	BeforeAll(func() {
		if os.Getuid() != 0 {
			Skip("needs root")
		}
	})

	It("returns our own hostname", func() {
		Expect(hostname(0)).To(Equal(Successful(os.Hostname())))
	})

	It("reads from other UTS namespace", func(ctx context.Context) {
		By("creating a new Docker session for testing")
		sess := Successful(morbyd.NewSession(ctx,
			session.WithAutoCleaning("test.whalewatcher=engineclient/cri")))
		DeferCleanup(func(ctx context.Context) {
			By("auto-cleaning the session")
			sess.Close(ctx)
		})

		By("creating a canary container")
		canaryCntr := Successful(sess.Run(ctx,
			"busybox",
			run.WithName(testName),
			run.WithAutoRemove(),
			run.WithHostname(testHostname),
			run.WithCommand(
				"/bin/sh",
				"-c",
				"mkdir -p /www && echo Hellorld!>/www/index.html && "+
					"httpd -f -p 5099 -h /www",
			),
		))

		By("visiting other UTS and reading its hostname")
		pid := Successful(canaryCntr.PID(ctx))
		visitUTS(pid, func() {
			GinkgoHelper()
			Expect(os.Hostname()).To(Equal(testHostname))
		})
		Expect(os.Hostname()).NotTo(Equal(testHostname), "UTS namespace spill-over")

		By("calling our hostname()")
		Expect(hostname(pid)).To(Equal(testHostname))

		By("removing the UTS hostname")
		visitUTS(pid, func() {
			GinkgoHelper()
			Expect(syscall.Sethostname([]byte(""))).To(Succeed())
		})
		By("calling our hostname()")
		Expect(hostname(pid)).To(Equal(testHostname))
	})

})
