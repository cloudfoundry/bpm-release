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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cgroups", func() {
	Describe("finding enabled cgroup subsystems", func() {
		It("shows enabled subsystems from /proc/cgroups", func() {
			r := strings.NewReader(`#subsys_name	hierarchy	num_cgroups	enabled
cpuset	11	2	1
cpu	8	101	1
cpuacct	8	101	0
blkio	10	101	1
memory	6	298	1
devices	9	101	1
freezer	12	2	1
net_cls	3	2	0
perf_event	4	2	1
net_prio	3	2	1
hugetlb	5	2	0
pids	7	108	1
rdma	2	1	1`)
			systems, err := enabledSubsystems(r)
			Expect(err).NotTo(HaveOccurred())

			Expect(systems).To(ConsistOf("cpuset", "cpu", "blkio", "memory", "devices", "freezer", "perf_event", "net_prio", "pids", "rdma"))
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
			group, err := subsystemGrouping(r, "memory")
			Expect(err).ToNot(HaveOccurred())
			Expect(group).To(Equal("memory"))
		})

		It("handles grouped subsystems", func() {
			group, err := subsystemGrouping(r, "cpu")
			Expect(err).ToNot(HaveOccurred())
			Expect(group).To(Equal("cpu,cpuacct"))
		})
	})
})
