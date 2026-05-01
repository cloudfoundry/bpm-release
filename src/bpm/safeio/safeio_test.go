// Copyright (C) 2026-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package safeio_test

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"golang.org/x/sys/unix"

	"bpm/safeio"
)

var _ = Describe("OpenAppendChown", func() {
	var (
		tmpDir   string
		uid, gid int
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "safeio")
		Expect(err).NotTo(HaveOccurred())

		uid = os.Getuid()
		gid = os.Getgid()
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Context("when the path does not yet exist and the parent is a regular directory", func() {
		It("creates the file with the requested mode and ownership", func() {
			path := filepath.Join(tmpDir, "fresh.log")

			f, err := safeio.OpenAppendChown(path, uid, gid, 0600)
			Expect(err).NotTo(HaveOccurred())
			defer f.Close() //nolint:errcheck

			info, err := f.Stat()
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModePerm).To(Equal(os.FileMode(0600)))
			Expect(info.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(uid)))
			Expect(info.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(gid)))
		})
	})

	Context("when the path already exists as a regular file", func() {
		It("opens the existing file for append without truncating it", func() {
			path := filepath.Join(tmpDir, "existing.log")
			Expect(os.WriteFile(path, []byte("first\n"), 0600)).To(Succeed())

			f, err := safeio.OpenAppendChown(path, uid, gid, 0600)
			Expect(err).NotTo(HaveOccurred())
			defer f.Close() //nolint:errcheck

			_, err = f.WriteString("second\n")
			Expect(err).NotTo(HaveOccurred())

			contents, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("first\nsecond\n"))
		})
	})

	Context("when the leaf path is a symlink to an existing file", func() {
		var (
			target string
			link   string
		)

		BeforeEach(func() {
			target = filepath.Join(tmpDir, "target.txt")
			link = filepath.Join(tmpDir, "link.log")

			Expect(os.WriteFile(target, []byte("untouched"), 0600)).To(Succeed())
			Expect(os.Symlink(target, link)).To(Succeed())
		})

		It("returns an ELOOP error and does not modify the target", func() {
			_, err := safeio.OpenAppendChown(link, uid, gid, 0600)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, unix.ELOOP)).To(BeTrue(), "expected error to wrap unix.ELOOP, got %v", err)

			contents, err := os.ReadFile(target)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("untouched"))
		})
	})

	Context("when the leaf path is a symlink to a non-existent target", func() {
		var (
			target string
			link   string
		)

		BeforeEach(func() {
			target = filepath.Join(tmpDir, "missing-target.txt")
			link = filepath.Join(tmpDir, "link.log")

			Expect(os.Symlink(target, link)).To(Succeed())
		})

		It("returns an ELOOP error and does not create the target", func() {
			_, err := safeio.OpenAppendChown(link, uid, gid, 0600)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, unix.ELOOP)).To(BeTrue(), "expected error to wrap unix.ELOOP, got %v", err)

			Expect(target).NotTo(BeAnExistingFile())
		})
	})

	Context("when the parent directory does not exist", func() {
		It("returns an ENOENT-style error and does not create anything", func() {
			path := filepath.Join(tmpDir, "missing-dir", "file.log")

			_, err := safeio.OpenAppendChown(path, uid, gid, 0600)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue(), "expected NotExist error, got %v", err)
			Expect(path).NotTo(BeAnExistingFile())
		})
	})
})
