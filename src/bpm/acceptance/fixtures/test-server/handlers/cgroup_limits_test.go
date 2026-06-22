// Copyright (C) 2017-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package handlers

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handlers Suite")
}

var _ = Describe("findCgroupDir", func() {
	var root string

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "cgroup-test-*")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(root)).To(Succeed())
	})

	Context("legacy systemd mode (runc default scope name)", func() {
		It("finds runc-bpm-test-server.scope", func() {
			scopeDir := filepath.Join(root, "system.slice", "runc-bpm-test-server.scope")
			Expect(os.MkdirAll(scopeDir, 0755)).To(Succeed())

			found, err := findCgroupDir(root, "bpm-test-server")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(Equal(scopeDir))
		})
	})

	Context("cgroup-v2-aware systemd mode (scope name from ToSystemdCgroupsPath)", func() {
		It("finds a scope with a warden garden prefix ending in -bpm-test-server.scope", func() {
			scopeDir := filepath.Join(root, "system.slice", "garden-abc-scope-bpm-bpm-test-server.scope")
			Expect(os.MkdirAll(scopeDir, 0755)).To(Succeed())

			found, err := findCgroupDir(root, "bpm-test-server")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(Equal(scopeDir))
		})

		It("finds a scope with a monit-service prefix", func() {
			scopeDir := filepath.Join(root, "system.slice", "bpm-service-bpm-bpm-test-server.scope")
			Expect(os.MkdirAll(scopeDir, 0755)).To(Succeed())

			found, err := findCgroupDir(root, "bpm-test-server")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(Equal(scopeDir))
		})
	})

	Context("cgroupfs mode (cgroup v2 without systemd)", func() {
		It("finds a directory named exactly containerID", func() {
			cgroupDir := filepath.Join(root, "docker", "abc123", "bpm-test-server")
			Expect(os.MkdirAll(cgroupDir, 0755)).To(Succeed())

			found, err := findCgroupDir(root, "bpm-test-server")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(Equal(cgroupDir))
		})
	})

	Context("named process", func() {
		It("finds the scope for a named process using the .2e encoding", func() {
			scopeDir := filepath.Join(root, "system.slice", "runc-bpm-test-server.2ealt-server.scope")
			Expect(os.MkdirAll(scopeDir, 0755)).To(Succeed())

			found, err := findCgroupDir(root, "bpm-test-server.2ealt-server")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(Equal(scopeDir))
		})
	})

	Context("not found", func() {
		It("returns an error when no matching directory exists", func() {
			_, err := findCgroupDir(root, "bpm-test-server")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("bpm-test-server"))
		})

		It("does not match a directory that only partially matches the container ID", func() {
			wrongDir := filepath.Join(root, "bpm-test-server-extra")
			Expect(os.MkdirAll(wrongDir, 0755)).To(Succeed())

			_, err := findCgroupDir(root, "bpm-test-server")
			Expect(err).To(HaveOccurred())
		})

		It("does not match a scope for a different container", func() {
			wrongScope := filepath.Join(root, "system.slice", "runc-bpm-other-server.scope")
			Expect(os.MkdirAll(wrongScope, 0755)).To(Succeed())

			_, err := findCgroupDir(root, "bpm-test-server")
			Expect(err).To(HaveOccurred())
		})
	})
})
