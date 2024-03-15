// Copyright (C) 2017-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package client_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/runtime-spec/specs-go"

	"bpm/runc/client"
)

var _ = Describe("RuncClient", func() {
	var (
		runcClient *client.RuncClient
		jobSpec    specs.Spec
		bundlePath string
		user       specs.User
	)

	BeforeEach(func() {
		user = specs.User{UID: 200, GID: 300, Username: "vcap"}
		runcClient = client.NewRuncClient(
			"/var/vcap/packages/runc/bin/runc",
			"/var/vcap/data/bpm/runc",
			false,
		)
	})

	Describe("CreateBundle", func() {
		var bundlesRoot string

		BeforeEach(func() {
			jobSpec = specs.Spec{
				Version: "example-version",
			}

			var err error
			bundlesRoot, err = os.MkdirTemp("", "bundle-builder")
			Expect(err).ToNot(HaveOccurred())

			bundlePath = filepath.Join(bundlesRoot, "bundle")
		})

		AfterEach(func() {
			Expect(os.RemoveAll(bundlesRoot)).To(Succeed())
		})

		It("makes the bundle directory", func() {
			err := runcClient.CreateBundle(bundlePath, jobSpec, user)
			Expect(err).ToNot(HaveOccurred())

			f, err := os.Stat(bundlePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
		})

		It("makes an empty rootfs directory", func() {
			err := runcClient.CreateBundle(bundlePath, jobSpec, user)
			Expect(err).ToNot(HaveOccurred())

			bundlefs := filepath.Join(bundlePath, "rootfs")
			f, err := os.Stat(bundlefs)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0755)))
			Expect(f.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
			Expect(f.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(0)))

			infos, err := os.ReadDir(bundlefs)
			Expect(err).ToNot(HaveOccurred())
			Expect(infos).To(HaveLen(0))
		})

		It("writes a config.json in the root bundle directory", func() {
			err := runcClient.CreateBundle(bundlePath, jobSpec, user)
			Expect(err).ToNot(HaveOccurred())

			configPath := filepath.Join(bundlePath, "config.json")
			f, err := os.Stat(configPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0600)))

			expectedConfigData, err := json.MarshalIndent(&jobSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			configData, err := os.ReadFile(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(configData).To(MatchJSON(expectedConfigData))
		})

		Context("when creating the bundle directory fails", func() {
			BeforeEach(func() {
				_, err := os.Create(bundlePath)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				err := runcClient.CreateBundle(bundlePath, jobSpec, user)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the rootfs directory fails", func() {
			BeforeEach(func() {
				err := os.MkdirAll(bundlePath, 0700)
				Expect(err).ToNot(HaveOccurred())

				_, err = os.Create(filepath.Join(bundlePath, "rootfs"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				err := runcClient.CreateBundle(bundlePath, jobSpec, user)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("DestroyBundle", func() {
		var bundlePath string

		BeforeEach(func() {
			var err error
			bundlePath, err = os.MkdirTemp("", "bundle-builder")
			Expect(err).ToNot(HaveOccurred())

			jobSpec := specs.Spec{
				Version: "test-version",
			}
			user := specs.User{Username: "vcap", UID: 300, GID: 400}

			err = runcClient.CreateBundle(bundlePath, jobSpec, user)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(bundlePath)).To(Succeed())
		})

		It("deletes the bundle", func() {
			err := runcClient.DestroyBundle(bundlePath)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(bundlePath)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Describe("ListContainers", func() {
		var (
			tempDir      string
			fakeRuncPath string
		)

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "")
			Expect(err).NotTo(HaveOccurred())

			fakeRuncPath = filepath.Join(tempDir, "fakeRunc")

			runcClient = client.NewRuncClient(fakeRuncPath, "/path/to/things", false)
		})

		AfterEach(func() {
			err := os.RemoveAll(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})

		// RunC has a race condition where it will try and put a container in
		// the list before it can fetch the state for that container which
		// dumps errors in stderr.
		Context("when runc returns an error as well as a list", func() {
			BeforeEach(func() {
				contents := []byte(`#!/bin/sh
echo -n 'error: could not list' >&2
echo -n '[]'
exit 0
`)

				err := os.WriteFile(fakeRuncPath, contents, 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			It("ignores the error", func() {
				containers, err := runcClient.ListContainers()
				Expect(err).NotTo(HaveOccurred())
				Expect(containers).To(BeEmpty())
			})
		})
	})

	Context("when running in systemd", func() {
		var (
			tempDir      string
			fakeRuncPath string
		)

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "")
			Expect(err).NotTo(HaveOccurred())

			fakeRuncPath = filepath.Join(tempDir, "fakeRunc")
			contents := []byte(`#!/bin/sh

echo "{}"

if echo "$@" | grep -q -- "--systemd-cgroup"; then
  exit 0
else
  exit 1
fi
`)

			err = os.WriteFile(fakeRuncPath, contents, 0700)
			Expect(err).NotTo(HaveOccurred())

			runcClient = client.NewRuncClient(fakeRuncPath, "/path/to/things", true)
		})

		AfterEach(func() {
			err := os.RemoveAll(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("passes the --systemd-cgroup flag to runc", func() {
			_, err := runcClient.ContainerState("foo")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("ContainerState", func() {
		var (
			tempDir      string
			fakeRuncPath string
		)

		BeforeEach(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "")
			Expect(err).NotTo(HaveOccurred())

			fakeRuncPath = filepath.Join(tempDir, "fakeRunc")

			runcClient = client.NewRuncClient(fakeRuncPath, "/path/to/things", false)
		})

		AfterEach(func() {
			err := os.RemoveAll(tempDir)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when the error message indicates the container is not running", func() {
			BeforeEach(func() {
				contents := []byte(`#!/bin/sh
echo -n '{"msg": "container does not exist"}'
exit 1
`)

				err := os.WriteFile(fakeRuncPath, contents, 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns nil,nil", func() {
				state, err := runcClient.ContainerState("foo")
				Expect(err).NotTo(HaveOccurred())

				Expect(state).To(BeNil())
			})

			Context("when the error message contains spaces", func() {
				BeforeEach(func() {
					// Note the echo also purposefully prints a newline as well as spaces
					contents := []byte(`#!/bin/sh
echo '         {"msg":"container does not exist"}     '
exit 1
`)

					err := os.WriteFile(fakeRuncPath, contents, 0700)
					Expect(err).NotTo(HaveOccurred())
				})

				It("strips spaces from the error message", func() {
					state, err := runcClient.ContainerState("foo")
					Expect(err).NotTo(HaveOccurred())

					Expect(state).To(BeNil())
				})
			})
		})

		Context("when the error message contains other information", func() {
			BeforeEach(func() {
				contents := []byte(`#!/bin/sh
echo -n 'some unrelated error'
exit 1
`)

				err := os.WriteFile(fakeRuncPath, contents, 0700)
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns nil,nil", func() {
				state, err := runcClient.ContainerState("foo")
				Expect(err).To(HaveOccurred())

				Expect(state).To(BeNil())
			})
		})
	})
})
