// Copyright (C) 2019-Present CloudFoundry.org Foundation, Inc. All rights reserved.
//
// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License”);
// you may not use this file except in compliance with the License.
//
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
// License for the specific language governing permissions and limitations
// under the License.

package hostlock_test

import (
	"io/ioutil"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bpm/hostlock"
)

var _ = Describe("Hostlock", func() {
	var tmpdir string

	BeforeEach(func() {
		var err error
		tmpdir, err = ioutil.TempDir("", "hostlock_test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpdir)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("locking jobs", func() {
		It("does not allow two of the same lock to be held at once", func() {
			locks := hostlock.NewHandle(tmpdir)

			held, err := locks.LockJob("job", "process")
			Expect(err).NotTo(HaveOccurred())

			c := make(chan struct{})

			go func() {
				otherHeld, err := locks.LockJob("job", "process")
				Expect(err).NotTo(HaveOccurred())

				close(c)

				err = otherHeld.Unlock()
				Expect(err).NotTo(HaveOccurred())
			}()

			Consistently(c).ShouldNot(BeClosed())

			err = held.Unlock()
			Expect(err).NotTo(HaveOccurred())

			Eventually(c).Should(BeClosed())
		})

		It("allows two different locks to be held at once", func() {
			locks := hostlock.NewHandle(tmpdir)

			held, err := locks.LockJob("job", "process")
			Expect(err).NotTo(HaveOccurred())

			c := make(chan struct{})

			go func() {
				otherHeld, err := locks.LockJob("other", "process")
				Expect(err).NotTo(HaveOccurred())

				close(c)

				err = otherHeld.Unlock()
				Expect(err).NotTo(HaveOccurred())
			}()

			Eventually(c).Should(BeClosed())

			err = held.Unlock()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
