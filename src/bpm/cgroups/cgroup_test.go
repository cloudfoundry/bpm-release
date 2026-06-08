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

package cgroups

import (
	"io"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cgroups", func() {
	Describe("SelfCgroupPath", func() {
		It("returns the cgroup v2 path from a valid unified-mode entry", func() {
			r := strings.NewReader("0::/garden/abc-123/\n")
			path, err := selfCgroupPathFromReader(r)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal("/garden/abc-123/"))
		})

		It("strips carriage return from CRLF line endings", func() {
			r := strings.NewReader("0::/some/path\r\n")
			path, err := selfCgroupPathFromReader(r)
			Expect(err).NotTo(HaveOccurred())
			Expect(path).To(Equal("/some/path"))
		})

		It("errors when there is no unified-mode entry", func() {
			r := strings.NewReader("12:memory:/user.slice\n11:cpu:/user.slice\n")
			_, err := selfCgroupPathFromReader(r)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("no cgroup v2 entry")))
		})

		It("errors on empty input", func() {
			r := strings.NewReader("")
			_, err := selfCgroupPathFromReader(r)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("checking subsystem grouping", func() {
		var r io.Reader

		BeforeEach(func() {
			r = strings.NewReader(`12:devices:/system.slice/runit.service/aa4575c9-58b0-4f62-540e-7bd137e5170f
11:cpu,cpuacct:/aa4575c9-58b0-4f62-540e-7bd137e5170f
10:rdma:/
9:cpuset:/aa4575c9-58b0-4f62-540e-7bd137e5170f
8:freezer:/aa4575c9-58b0-4f62-540e-7bd137e5170f
7:perf_event:/aa4575c9-58b0-4f62-540e-7bd137e5170f
6:net_cls,net_prio:/aa4575c9-58b0-4f62-540e-7bd137e5170f
5:hugetlb:/aa4575c9-58b0-4f62-540e-7bd137e5170f
4:blkio:/aa4575c9-58b0-4f62-540e-7bd137e5170f
3:memory:/aa4575c9-58b0-4f62-540e-7bd137e5170f
2:pids:/aa4575c9-58b0-4f62-540e-7bd137e5170f
1:name=systemd:/system.slice/runit.service/aa4575c9-58b0-4f62-540e-7bd137e5170f`)
		})

		It("handles singleton groups", func() {
			group, err := subsystemGroupingFromProcCgroup(r, "memory")
			Expect(err).ToNot(HaveOccurred())
			Expect(group).To(Equal("memory"))
		})

		It("handles grouped subsystems", func() {
			group, err := subsystemGroupingFromProcCgroup(r, "cpu")
			Expect(err).ToNot(HaveOccurred())
			Expect(group).To(Equal("cpu,cpuacct"))
		})
	})

	Describe("ToSystemdCgroupsPath", func() {
		It("converts a nested garden scope path", func() {
			Expect(ToSystemdCgroupsPath("/system.slice/garden-abc.scope/monit.service", "bpm-uaa")).
				To(Equal("system.slice:garden-abc-scope-bpm:bpm-uaa"))
		})

		It("uses slice name for uniqueness when path has no intermediate scope", func() {
			Expect(ToSystemdCgroupsPath("/system.slice", "bpm-uaa")).
				To(Equal("system.slice:system-slice-bpm:bpm-uaa"))
		})

		It("uses first path element for uniqueness when no .slice component found", func() {
			Expect(ToSystemdCgroupsPath("/garden-abc.scope", "bpm-uaa")).
				To(Equal("system.slice:garden-abc-scope-bpm:bpm-uaa"))
		})
	})
})
