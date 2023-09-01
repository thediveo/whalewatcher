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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/thediveo/success"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

var _ = Describe("CRI client", func() {

	It("successfully fetches information about a CRI API service provider", func(ctx context.Context) {
		if os.Getuid() != 0 {
			Skip("needs root")
		}

		cl := Successful(New(defaultCRIEndpoint))
		defer func() {
			Expect(cl.Close()).To(Succeed())
		}()

		Expect(cl.rtcl.Version(ctx, &v1.VersionRequest{})).To(SatisfyAll(
			HaveField("Version", Not(BeEmpty())),
			HaveField("RuntimeName", Not(BeEmpty())),
		))
	})

})
