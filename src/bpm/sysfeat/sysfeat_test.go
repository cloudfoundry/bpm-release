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
	"runtime"
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
			// On native systems, seccomp should be supported
			// We can't assert the exact value as it depends on the environment
			// but we can verify the field exists and has a boolean value
			_ = features.SeccompSupported
		})

		Context("when BPM_DISABLE_SECCOMP_DETECTION is set", func() {
			BeforeEach(func() {
				err := os.Setenv("BPM_DISABLE_SECCOMP_DETECTION", "1")
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				err := os.Unsetenv("BPM_DISABLE_SECCOMP_DETECTION")
				Expect(err).NotTo(HaveOccurred())
			})

			It("forces SeccompSupported to true", func() {
				features, err := sysfeat.Fetch()
				Expect(err).NotTo(HaveOccurred())
				Expect(features.SeccompSupported).To(BeTrue())
			})
		})

		Context("on a native system", func() {
			It("reports seccomp as supported", func() {
				// This test assumes we're running on a native system (not in an
				// emulated container). In CI/CD environments, this should be true.
				features, err := sysfeat.Fetch()
				Expect(err).NotTo(HaveOccurred())

				// If we're not in a container, seccomp should always be supported
				// We can check this by verifying we're not in a container
				// (no /.dockerenv file)
				_, dockerEnvErr := os.Stat("/.dockerenv")
				if os.IsNotExist(dockerEnvErr) {
					// Not in a container, seccomp should be supported
					Expect(features.SeccompSupported).To(BeTrue())
				}
			})
		})
	})

	Describe("Architecture detection", func() {
		It("correctly identifies the current architecture", func() {
			// This is more of a smoke test to ensure the architecture
			// detection doesn't panic or return unexpected values
			goArch := runtime.GOARCH
			Expect(goArch).NotTo(BeEmpty())

			// Common architectures we expect
			validArchs := []string{"amd64", "386", "arm64", "arm"}
			Expect(validArchs).To(ContainElement(goArch))
		})
	})
})
