// Copyright (C) 2018-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package mount

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Mount", func() {
	Describe("Mounts", func() {
		It("parses the contents of /proc/mounts", func() {
			mnts, err := Mounts()
			Expect(err).NotTo(HaveOccurred())
			Expect(mnts).ToNot(BeEmpty())
		})
	})

	Describe("parseMountFile", func() {
		It("returns a slice of mounts", func() {
			mnts, err := parseMountFile("fixtures/mount")
			Expect(err).NotTo(HaveOccurred())
			Expect(mnts).To(ConsistOf(
				Mnt{
					Device:     "proc",
					MountPoint: "/proc",
					Filesystem: "proc",
					Options:    []string{"rw", "nosuid", "nodev", "noexec", "relatime"},
				},
				Mnt{
					Device:     "tmpfs",
					MountPoint: "/dev",
					Filesystem: "tmpfs",
					Options:    []string{"rw", "nosuid", "size=65536k", "mode=755"},
				},
				Mnt{
					Device:     "devpts",
					MountPoint: "/dev/console",
					Filesystem: "devpts",
					Options:    []string{"rw", "nosuid", "noexec", "relatime", "gid=5", "mode=620", "ptmxmode=666"},
				},
			))
		})

		Context("when the file contains an invalid mount format", func() {
			It("returns an error", func() {
				_, err := parseMountFile("fixtures/invalid-mount")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the file doesn't exist", func() {
			It("returns an error", func() {
				_, err := parseMountFile("this-is-non-existant")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
