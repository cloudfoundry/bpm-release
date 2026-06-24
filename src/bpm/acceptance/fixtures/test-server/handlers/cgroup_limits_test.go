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
	"encoding/json"
	"flag"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	// ginkgo -r (recursive) passes all flags from the outer acceptance suite to
	// every discovered sub-package. Define them here (unused) so that
	// flag.Parse() does not fail with "flag provided but not defined".
	flag.String("agent-uri", "", "unused in handlers unit tests")
	flag.String("observer-uri", "", "unused in handlers unit tests")
}

func TestHandlers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Handlers Suite")
}

var _ = Describe("CgroupLimits path traversal guard", func() {
	callHandler := func(cgroupPath string) cgroupLimitsResponse {
		req := httptest.NewRequest(http.MethodGet, "/cgroup-limits?cgroup-path="+cgroupPath, nil)
		rec := httptest.NewRecorder()
		CgroupLimits(rec, req)
		var resp cgroupLimitsResponse
		Expect(json.NewDecoder(rec.Body).Decode(&resp)).To(Succeed())
		return resp
	}

	DescribeTable("rejects paths that escape /sys/fs/cgroup",
		func(traversal string) {
			resp := callHandler(traversal)
			Expect(resp.Error).To(ContainSubstring("escapes the cgroup root"),
				"path %q should be rejected", traversal)
		},
		Entry("one level above cgroup root", "/../etc"),
		Entry("three levels above cgroup root", "/../../../etc"),
		Entry("traversal starting inside subtree", "/docker/abc/../../../../etc"),
	)

	It("does not reject a path that stays inside the cgroup root via ..", func() {
		resp := callHandler("/docker/abc/../../etc")
		Expect(resp.Error).NotTo(ContainSubstring("escapes"))
	})

	It("accepts a well-formed absolute path inside the cgroup root", func() {
		// The path won't exist on the test host so memory.max will error, but
		// the traversal guard must not reject it.
		resp := callHandler("/docker/abc123/system.slice/monit-service-bpm-bpm-test-server.scope")
		Expect(resp.Error).NotTo(ContainSubstring("escapes"))
	})
})

var _ = Describe("parseCgroupV2Path", func() {
	It("returns the cgroup v2 unified-mode path", func() {
		input := strings.NewReader("12:blkio:/\n0::/docker/abc123/system.slice/monit.service\n")
		path, err := parseCgroupV2Path(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal("/docker/abc123/system.slice/monit.service"))
	})

	It("ignores v1 hierarchy entries and returns the 0:: entry", func() {
		input := strings.NewReader(
			"11:memory:/docker/abc123\n" +
				"10:cpu,cpuacct:/docker/abc123\n" +
				"0::/docker/abc123/system.slice/monit-service-bpm-bpm-test-server.scope\n",
		)
		path, err := parseCgroupV2Path(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal("/docker/abc123/system.slice/monit-service-bpm-bpm-test-server.scope"))
	})

	It("returns an error when there is no 0:: entry", func() {
		input := strings.NewReader("11:memory:/foo\n10:cpu:/foo\n")
		_, err := parseCgroupV2Path(input)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("0::"))
	})

	It("returns an error for an empty file", func() {
		_, err := parseCgroupV2Path(strings.NewReader(""))
		Expect(err).To(HaveOccurred())
	})

	It("handles a pure cgroup v2 system where 0:: is the only entry", func() {
		input := strings.NewReader("0::/user.slice/user-1000.slice/session-1.scope\n")
		path, err := parseCgroupV2Path(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal("/user.slice/user-1000.slice/session-1.scope"))
	})

	It("handles a path of / (container at the cgroup root)", func() {
		input := strings.NewReader("0::/\n")
		path, err := parseCgroupV2Path(input)
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal("/"))
	})
})
