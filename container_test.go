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
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math/rand"

	"github.com/thediveo/whalewatcher/test/mockingmoby"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// newTestContainer adds a new fake/mock container with the specified name and
// project name label, as well as a random ID string. The container ID and PID
// is then returned to the caller.
func newTestContainer(mm *mockingmoby.MockingMoby, name, projectname string) (string, int) {
	o := make([]byte, 32) // length of fake SHA256 in "octets" :p
	_, err := crand.Read(o)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	id := hex.EncodeToString(o)
	pid := rand.Intn(4194303) + 1
	mm.AddContainer(mockingmoby.MockedContainer{
		ID:     id,
		Name:   name,
		Status: mockingmoby.MockedRunning,
		PID:    pid,
		Labels: map[string]string{
			ComposerProjectLabel: projectname,
		},
	})
	return id, pid
}

var _ = Describe("container proxy", func() {

	var mm *mockingmoby.MockingMoby

	BeforeEach(func() {
		mm = mockingmoby.NewMockingMoby()
		Expect(mm).NotTo(BeNil())
	})

	AfterEach(func() {
		mm.Close()
	})

	It("stringifies", func() {
		pp, pppid := newTestContainer(mm, "poehser_puhbe", "gnampf")
		c, err := newContainer(context.Background(), mm, pp)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.String()).To(MatchRegexp(
			fmt.Sprintf(`container '%s'/%s from project 'gnampf' with PID %d`,
				"poehser_puhbe", pp, pppid)))
	})

	It("fails for invalid id/name", func() {
		c, err := newContainer(context.Background(), mm, "rusty_rumpelpumpel")
		Expect(err).To(HaveOccurred())
		Expect(c).To(BeNil())
	})

	It("fails when stopped", func() {
		pp, _ := newTestContainer(mm, "poehser_puhbe", "gnampf")
		// Only stop, but don't remove the fake container yet.
		mm.StopContainer(pp)
		c, err := newContainer(context.Background(), mm, "poehser_puhbe")
		Expect(err).To(HaveOccurred())
		Expect(c).To(BeNil())
	})

	It("new from inspection", func() {
		pp, pppid := newTestContainer(mm, "poehser_puhbe", "gnampf")
		c, err := newContainer(context.Background(), mm, pp)
		Expect(err).NotTo(HaveOccurred())
		Expect(c.ID).To(Equal(pp))
		Expect(c.Name).To(Equal("poehser_puhbe"))
		Expect(c.PID).To(Equal(pppid))
		Expect(c.ProjectName()).To(Equal("gnampf"))
	})

})
