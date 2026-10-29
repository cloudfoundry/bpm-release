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

var _ = Describe("MkdirAllChown", func() {
	var (
		tmpDir   string
		uid, gid int
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "safeio-mkdir")
		Expect(err).NotTo(HaveOccurred())

		// MkdirAllChown rejects symlinks in every path component, and on some
		// platforms the temp root itself sits behind one.
		tmpDir, err = filepath.EvalSymlinks(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		uid = os.Getuid()
		gid = os.Getgid()
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Context("when the directory does not exist", func() {
		It("creates the directory with 0700 permissions and correct ownership", func() {
			dirPath := filepath.Join(tmpDir, "sub", "newdir")

			err := safeio.MkdirAllChown(dirPath, uid, gid)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(dirPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
			Expect(info.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(info.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(uid)))
			Expect(info.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(gid)))
		})
	})

	Context("when the directory already exists", func() {
		It("ensures ownership without error", func() {
			dirPath := filepath.Join(tmpDir, "existingdir")
			Expect(os.Mkdir(dirPath, 0700)).To(Succeed())

			err := safeio.MkdirAllChown(dirPath, uid, gid)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(dirPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})
	})

	Context("when the path is a symlink to an existing directory", func() {
		var (
			targetDir string
			linkPath  string
		)

		BeforeEach(func() {
			targetDir = filepath.Join(tmpDir, "target-dir")
			linkPath = filepath.Join(tmpDir, "symlink-dir")

			Expect(os.Mkdir(targetDir, 0755)).To(Succeed())
			Expect(os.Symlink(targetDir, linkPath)).To(Succeed())
		})

		It("refuses to follow the symlink and returns an error", func() {
			err := safeio.MkdirAllChown(linkPath, uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to create directory through symlink"))

			// Target directory should retain original permissions
			info, err := os.Lstat(targetDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModePerm).To(Equal(os.FileMode(0755)))
		})
	})

	Context("when the path is a symlink to a non-existent path", func() {
		var (
			targetDir string
			linkPath  string
		)

		BeforeEach(func() {
			targetDir = filepath.Join(tmpDir, "nonexistent-target")
			linkPath = filepath.Join(tmpDir, "dangling-link")

			Expect(os.Symlink(targetDir, linkPath)).To(Succeed())
		})

		It("refuses to follow the symlink and does not create the target", func() {
			err := safeio.MkdirAllChown(linkPath, uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to create directory through symlink"))

			_, err = os.Lstat(targetDir)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Context("when the path exists as a regular file", func() {
		It("returns an error", func() {
			filePath := filepath.Join(tmpDir, "regular-file")
			Expect(os.WriteFile(filePath, []byte("data"), 0600)).To(Succeed())

			err := safeio.MkdirAllChown(filePath, uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("exists and is not a directory"))
		})
	})

	Context("when an intermediate component is a symlink", func() {
		var protectedDir string

		BeforeEach(func() {
			protectedDir = filepath.Join(tmpDir, "protected")
			Expect(os.MkdirAll(protectedDir, 0755)).To(Succeed())

			writable := filepath.Join(tmpDir, "writable")
			Expect(os.MkdirAll(writable, 0777)).To(Succeed())
			Expect(os.Chmod(writable, 0777)).To(Succeed())
			Expect(os.Symlink(protectedDir, filepath.Join(writable, "a"))).To(Succeed())
		})

		It("refuses to follow the symlink and does not create the leaf under the target", func() {
			err := safeio.MkdirAllChown(filepath.Join(tmpDir, "writable", "a", "b"), uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to create directory through symlink"))

			_, statErr := os.Lstat(filepath.Join(protectedDir, "b"))
			Expect(os.IsNotExist(statErr)).To(BeTrue(), "leaf was created through the intermediate symlink")
		})
	})

	Context("when the path is relative", func() {
		It("returns an error", func() {
			err := safeio.MkdirAllChown("relative/path", uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("relative path"))
		})
	})

	// The BOSH root is configurable (bosh.Path), and depending on stemcell and
	// release any of sys, sys/log or sys/run may be a root-owned symlink into
	// /var/vcap/data. These specs model that layout under the temp root rather
	// than touching the real /var/vcap
	Describe("trusted intermediate symlinks", func() {
		BeforeEach(func() {
			if os.Getuid() != 0 {
				Skip("must run as root to control the ownership the trust check keys off")
			}
		})

		It("follows a root-owned symlink in a root-owned, non-writable parent", func() {
			target := filepath.Join(tmpDir, "data", "sys", "log")
			Expect(os.MkdirAll(target, 0755)).To(Succeed())

			sysDir := filepath.Join(tmpDir, "sys")
			Expect(os.MkdirAll(sysDir, 0755)).To(Succeed())
			Expect(os.Symlink(target, filepath.Join(sysDir, "log"))).To(Succeed())

			jobDir := filepath.Join(sysDir, "log", "test-job")
			Expect(safeio.MkdirAllChown(jobDir, uid, gid)).To(Succeed())

			info, err := os.Lstat(filepath.Join(target, "test-job"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})

		It("follows a root-owned symlink at the sys component itself", func() {
			target := filepath.Join(tmpDir, "data", "sys")
			Expect(os.MkdirAll(filepath.Join(target, "log"), 0755)).To(Succeed())
			Expect(os.Symlink(target, filepath.Join(tmpDir, "sys"))).To(Succeed())

			jobDir := filepath.Join(tmpDir, "sys", "log", "test-job")
			Expect(safeio.MkdirAllChown(jobDir, uid, gid)).To(Succeed())

			info, err := os.Lstat(filepath.Join(target, "log", "test-job"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.IsDir()).To(BeTrue())
		})

		It("refuses a root-owned symlink whose parent is group-writable", func() {
			target := filepath.Join(tmpDir, "host-sensitive")
			Expect(os.MkdirAll(target, 0700)).To(Succeed())

			writable := filepath.Join(tmpDir, "writable")
			Expect(os.MkdirAll(writable, 0775)).To(Succeed())
			Expect(os.Chmod(writable, 0775)).To(Succeed())
			Expect(os.Symlink(target, filepath.Join(writable, "sub"))).To(Succeed())

			err := safeio.MkdirAllChown(filepath.Join(writable, "sub", "job"), uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to create directory through symlink"))

			_, statErr := os.Lstat(filepath.Join(target, "job"))
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})

		It("refuses a symlink not owned by root", func() {
			target := filepath.Join(tmpDir, "host-sensitive")
			Expect(os.MkdirAll(target, 0700)).To(Succeed())

			parent := filepath.Join(tmpDir, "parent")
			Expect(os.MkdirAll(parent, 0755)).To(Succeed())
			link := filepath.Join(parent, "sub")
			Expect(os.Symlink(target, link)).To(Succeed())
			Expect(os.Lchown(link, 1000, 1000)).To(Succeed())

			err := safeio.MkdirAllChown(filepath.Join(parent, "sub", "job"), uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to create directory through symlink"))

			_, statErr := os.Lstat(filepath.Join(target, "job"))
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})

		It("refuses a trusted-looking symlink at the leaf", func() {
			target := filepath.Join(tmpDir, "data", "job")
			Expect(os.MkdirAll(target, 0755)).To(Succeed())

			sysDir := filepath.Join(tmpDir, "sys")
			Expect(os.MkdirAll(sysDir, 0755)).To(Succeed())
			Expect(os.Symlink(target, filepath.Join(sysDir, "job"))).To(Succeed())

			err := safeio.MkdirAllChown(filepath.Join(sysDir, "job"), uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to create directory through symlink"))
		})
	})
})

var _ = Describe("Lchown", func() {
	var (
		tmpDir   string
		uid, gid int
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "safeio-lchown")
		Expect(err).NotTo(HaveOccurred())

		tmpDir, err = filepath.EvalSymlinks(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		uid = os.Getuid()
		gid = os.Getgid()
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Context("when the path is a regular file", func() {
		It("successfully changes ownership", func() {
			filePath := filepath.Join(tmpDir, "file.txt")
			Expect(os.WriteFile(filePath, []byte("test"), 0600)).To(Succeed())

			err := safeio.Lchown(filePath, uid, gid)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(filePath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(uid)))
			Expect(info.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(gid)))
		})
	})

	Context("when the path is a symlink", func() {
		It("refuses to chown and returns an error without altering the target", func() {
			target := filepath.Join(tmpDir, "target.txt")
			link := filepath.Join(tmpDir, "link.txt")

			Expect(os.WriteFile(target, []byte("protected"), 0600)).To(Succeed())
			Expect(os.Symlink(target, link)).To(Succeed())

			err := safeio.Lchown(link, uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to chown symlink"))
		})
	})

	Context("when an intermediate component is a symlink", func() {
		It("refuses to chown without altering the target", func() {
			protectedDir := filepath.Join(tmpDir, "protected")
			Expect(os.MkdirAll(protectedDir, 0755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(protectedDir, "target.txt"), []byte("data"), 0600)).To(Succeed())

			writable := filepath.Join(tmpDir, "writable")
			Expect(os.MkdirAll(writable, 0777)).To(Succeed())
			Expect(os.Chmod(writable, 0777)).To(Succeed())
			Expect(os.Symlink(protectedDir, filepath.Join(writable, "sub"))).To(Succeed())

			err := safeio.Lchown(filepath.Join(writable, "sub", "target.txt"), uid, gid)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to chown symlink"))
		})
	})
})

var _ = Describe("Lchmod", func() {
	var (
		tmpDir string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "safeio-lchmod")
		Expect(err).NotTo(HaveOccurred())

		tmpDir, err = filepath.EvalSymlinks(tmpDir)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	Context("when the path is a directory", func() {
		It("changes permissions successfully", func() {
			dirPath := filepath.Join(tmpDir, "testdir")
			Expect(os.Mkdir(dirPath, 0755)).To(Succeed())

			err := safeio.Lchmod(dirPath, 0700)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(dirPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
		})
	})

	Context("when the path or an intermediate component is a symlink", func() {
		It("refuses to chmod and returns an error without altering the target", func() {
			targetDir := filepath.Join(tmpDir, "target")
			Expect(os.Mkdir(targetDir, 0755)).To(Succeed())
			link := filepath.Join(tmpDir, "link")
			Expect(os.Symlink(targetDir, link)).To(Succeed())

			err := safeio.Lchmod(link, 0700)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to chmod symlink"))

			info, err := os.Stat(targetDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModePerm).To(Equal(os.FileMode(0755)))
		})
	})
})
