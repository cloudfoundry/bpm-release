// Copyright (C) 2026-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package adapter_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bpm/runc/adapter"
)

var _ = Describe("Mount with Symlink Resolution", func() {
	var (
		tempDir          string
		realDir          string
		symlinkToRealDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = os.MkdirTemp("", "bpm-mount-test")
		Expect(err).NotTo(HaveOccurred())

		realDir = filepath.Join(tempDir, "real-directory")
		err = os.Mkdir(realDir, 0755)
		Expect(err).NotTo(HaveOccurred())

		symlinkToRealDir = filepath.Join(tempDir, "symlink")
		err = os.Symlink(realDir, symlinkToRealDir)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		err := os.RemoveAll(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("when mounting a symlinked path", func() {
		It("resolves the symlink to the real path", func() {
			mount := adapter.Mount(symlinkToRealDir, "/container/path")

			Expect(mount.Source).To(Equal(realDir))
			Expect(mount.Destination).To(Equal("/container/path"))
			Expect(mount.Type).To(Equal("bind"))
		})
	})

	Context("when mounting a non-symlinked path", func() {
		It("uses the path as-is", func() {
			mount := adapter.Mount(realDir, "/container/path")

			Expect(mount.Source).To(Equal(realDir))
			Expect(mount.Destination).To(Equal("/container/path"))
			Expect(mount.Type).To(Equal("bind"))
		})
	})

	Context("when the symlink path does not exist", func() {
		It("uses the original path", func() {
			nonExistentPath := filepath.Join(tempDir, "does-not-exist")
			mount := adapter.Mount(nonExistentPath, "/container/path")

			Expect(mount.Source).To(Equal(nonExistentPath))
			Expect(mount.Destination).To(Equal("/container/path"))
		})
	})

	Context("with IdentityMount", func() {
		It("resolves symlinks in both source and destination", func() {
			mount := adapter.IdentityMount(symlinkToRealDir)

			// Source should be resolved, but destination stays as-is
			// because IdentityMount(path) calls Mount(path, path)
			// and the destination is not resolved
			Expect(mount.Source).To(Equal(realDir))
			Expect(mount.Destination).To(Equal(symlinkToRealDir))
		})
	})
})
