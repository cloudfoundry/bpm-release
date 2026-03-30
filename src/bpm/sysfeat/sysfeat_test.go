// Copyright (C) 2018-Present CloudFoundry.org Foundation, Inc. All rights reserved.
//
// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License");
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

package sysfeat_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bpm/sysfeat"
)

func TestSysfeat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Sysfeat Suite")
}

var _ = Describe("Features", func() {
	Describe("Fetch", func() {
		It("returns a Features struct", func() {
			features, err := sysfeat.Fetch()
			Expect(err).NotTo(HaveOccurred())
			Expect(features).NotTo(BeNil())
		})

		It("includes SeccompSupported field", func() {
			features, err := sysfeat.Fetch()
			Expect(err).NotTo(HaveOccurred())
			_ = features.SeccompSupported
		})

		Context("when Rosetta binfmt_misc is not registered", func() {
			It("reports seccomp as supported", func() {
				_, err := os.Stat("/proc/sys/fs/binfmt_misc/rosetta")
				if !os.IsNotExist(err) {
					Skip("Rosetta binfmt_misc is registered on this host")
				}

				features, err := sysfeat.Fetch()
				Expect(err).NotTo(HaveOccurred())
				Expect(features.SeccompSupported).To(BeTrue())
			})
		})
	})
})
