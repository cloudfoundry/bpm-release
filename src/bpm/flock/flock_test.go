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

package flock_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bpm/flock"
)

var _ = Describe("Flock", func() {
	var tmpdir string

	BeforeEach(func() {
		var err error
		tmpdir, err = os.MkdirTemp("", "flocktest")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(tmpdir)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("an unlocked mutex", func() {
		var (
			lock     *flock.Flock
			lockPath string
		)

		BeforeEach(func() {
			lockPath = filepath.Join(tmpdir, "file.lock")

			var err error
			lock, err = flock.New(lockPath)
			Expect(err).NotTo(HaveOccurred())
		})

		It("cannot be unlocked again", func() {
			Expect(func() {
				lock.Unlock() //nolint:errcheck
			}).Should(Panic())
		})

		It("can be locked", func() {
			err := lock.Lock()
			Expect(err).NotTo(HaveOccurred())

			err = lock.Unlock()
			Expect(err).NotTo(HaveOccurred())
		})

		It("can be locked but not unlocked twice", func() {
			err := lock.Lock()
			Expect(err).NotTo(HaveOccurred())

			err = lock.Unlock()
			Expect(err).NotTo(HaveOccurred())

			Expect(func() {
				lock.Unlock() //nolint:errcheck
			}).Should(Panic())
		})

		Describe("protecting a shared resource", func() {
			var lock2 *flock.Flock

			BeforeEach(func() {
				var err error
				lock2, err = flock.New(lockPath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("does not allow concurrent access while the lock is held", func() {
				err := lock.Lock()
				Expect(err).NotTo(HaveOccurred())

				c := make(chan struct{})

				go func() {
					defer GinkgoRecover()

					err := lock2.Lock()
					Expect(err).NotTo(HaveOccurred())

					close(c)

					err = lock2.Unlock()
					Expect(err).NotTo(HaveOccurred())
				}()

				Consistently(c).ShouldNot(BeClosed())

				err = lock.Unlock()
				Expect(err).NotTo(HaveOccurred())

				Eventually(c).Should(BeClosed())
			})
		})
	})
})
