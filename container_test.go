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

package whalewatcher

import (
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// newTestContainer adds a new fake/mock container with the specified name and
// project name label, as well as a random ID string. The container ID and PID
// is then returned to the caller.
func newTestContainer(name, projectname string) (*Container, string, int) {
	o := make([]byte, 32) // length of fake SHA256 in "octets" :p
	_, err := crand.Read(o)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	id := hex.EncodeToString(o)
	pid := rand.Intn(4194303) + 1
	return &Container{
		ID:      id,
		Name:    name,
		PID:     pid,
		Project: projectname,
	}, id, pid
}

var _ = Describe("container proxy", func() {

	It("stringifies", func() {
		c, pp, pppid := newTestContainer("poehser_puhbe", "gnampf")
		Expect(c.String()).To(MatchRegexp(
			fmt.Sprintf(`container '%s'/%s from project 'gnampf' with PID %d`,
				"poehser_puhbe", pp, pppid)))
	})

})
