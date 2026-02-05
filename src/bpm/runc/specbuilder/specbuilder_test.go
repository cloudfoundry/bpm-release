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

package specbuilder_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"

	"bpm/runc/specbuilder"
)

func TestSpecbuilder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Specbuilder Suite")
}

var _ = Describe("SpecBuilder", func() {
	Describe("WithoutSeccomp", func() {
		var spec *specs.Spec

		BeforeEach(func() {
			spec = specbuilder.DefaultSpec()
		})

		It("removes seccomp from the spec", func() {
			Expect(spec.Linux.Seccomp).NotTo(BeNil())

			specbuilder.Apply(spec, specbuilder.WithoutSeccomp())

			Expect(spec.Linux.Seccomp).To(BeNil())
		})

		It("does not affect other security features", func() {
			// Capture original values
			originalNoNewPrivileges := spec.Process.NoNewPrivileges
			originalMaskedPaths := spec.Linux.MaskedPaths
			originalReadonlyPaths := spec.Linux.ReadonlyPaths
			originalUser := spec.Process.User

			specbuilder.Apply(spec, specbuilder.WithoutSeccomp())

			// Verify other security features are unchanged
			Expect(spec.Process.NoNewPrivileges).To(Equal(originalNoNewPrivileges))
			Expect(spec.Linux.MaskedPaths).To(Equal(originalMaskedPaths))
			Expect(spec.Linux.ReadonlyPaths).To(Equal(originalReadonlyPaths))
			Expect(spec.Process.User).To(Equal(originalUser))
		})

		It("does not affect capabilities", func() {
			// Add some capabilities
			specbuilder.Apply(spec, specbuilder.WithCapabilities([]string{"CAP_NET_BIND_SERVICE"}))
			
			originalCaps := spec.Process.Capabilities

			specbuilder.Apply(spec, specbuilder.WithoutSeccomp())

			// Verify capabilities are unchanged
			Expect(spec.Process.Capabilities).To(Equal(originalCaps))
		})

		It("does not affect user settings", func() {
			testUser := specs.User{UID: 1000, GID: 1000}
			specbuilder.Apply(spec, specbuilder.WithUser(testUser))

			specbuilder.Apply(spec, specbuilder.WithoutSeccomp())

			Expect(spec.Process.User).To(Equal(testUser))
		})

		It("does not affect mount options", func() {
			// Check that nosuid is still present on mounts
			var hasMountWithNosuid bool
			for _, mount := range spec.Mounts {
				for _, opt := range mount.Options {
					if opt == "nosuid" {
						hasMountWithNosuid = true
						break
					}
				}
			}

			Expect(hasMountWithNosuid).To(BeTrue(), "Expected at least one mount to have nosuid option")

			specbuilder.Apply(spec, specbuilder.WithoutSeccomp())

			// Verify nosuid is still present after WithoutSeccomp
			hasMountWithNosuid = false
			for _, mount := range spec.Mounts {
				for _, opt := range mount.Options {
					if opt == "nosuid" {
						hasMountWithNosuid = true
						break
					}
				}
			}

			Expect(hasMountWithNosuid).To(BeTrue(), "Expected nosuid to remain on mounts")
		})

		Context("when applied before WithPrivileged", func() {
			It("WithPrivileged still removes seccomp", func() {
				specbuilder.Apply(spec, specbuilder.WithoutSeccomp())
				Expect(spec.Linux.Seccomp).To(BeNil())

				specbuilder.Apply(spec, specbuilder.WithPrivileged())
				Expect(spec.Linux.Seccomp).To(BeNil())
			})
		})

		Context("when applied after WithPrivileged", func() {
			It("seccomp remains nil", func() {
				specbuilder.Apply(spec, specbuilder.WithPrivileged())
				Expect(spec.Linux.Seccomp).To(BeNil())

				specbuilder.Apply(spec, specbuilder.WithoutSeccomp())
				Expect(spec.Linux.Seccomp).To(BeNil())
			})
		})
	})

	Describe("DefaultSpec", func() {
		It("includes seccomp by default", func() {
			spec := specbuilder.DefaultSpec()
			Expect(spec.Linux.Seccomp).NotTo(BeNil())
			Expect(spec.Linux.Seccomp.Architectures).NotTo(BeEmpty())
			Expect(spec.Linux.Seccomp.Syscalls).NotTo(BeEmpty())
		})
	})
})
