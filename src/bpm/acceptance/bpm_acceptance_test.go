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

package bpm_acceptance_test

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BpmAcceptance", func() {
	BeforeEach(func() {
		if *agentURI == "" {
			Skip("-agent-uri must be provided")
		}
	})

	It("returns a 200 response with a body", func() {
		resp, err := client.Get(*agentURI)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("BPM is SWEET!\n"))
	})

	It("runs as the vcap user", func() {
		resp, err := client.Get(fmt.Sprintf("%s/whoami", *agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(Equal("vcap\n"))
	})

	It("has the correct bosh mounts", func() {
		resp, err := client.Get(fmt.Sprintf("%s/mounts", *agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		mounts := parseMounts(string(body))

		expectedMountPaths := map[string]string{
			"/var/vcap/data/packages":        "ro",
			"/var/vcap/data/test-server":     "rw",
			"/var/vcap/data/test-server/tmp": "rw",
			"/var/vcap/store/test-server":    "rw",
			"/var/vcap/jobs/test-server":     "ro",
			"/var/vcap/packages":             "ro",
			"/var/vcap/sys/log/test-server":  "rw",
		}

		var found []string
		for _, mount := range mounts {
			if strings.Contains(mount.path, "/var/vcap") {
				expectedOption, ok := expectedMountPaths[mount.path]
				Expect(ok).To(BeTrue(), fmt.Sprintf("found unexpected mount path %s", mount.path))

				found = append(found, mount.path)
				Expect(mount.options).To(ContainElement(expectedOption), fmt.Sprintf("no %s permissions for %s", expectedOption, mount.path))
			}
		}

		expectedLength := len(expectedMountPaths)
		Expect(found).To(HaveLen(expectedLength), fmt.Sprintf("missing mounts, actual: %#v, expected: %#v", found, expectedMountPaths))
	})

	It("has the correct read only system mounts", func() {
		resp, err := client.Get(fmt.Sprintf("%s/mounts", *agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		mounts := parseMounts(string(body))

		expectedMountPaths := []string{"/bin", "/etc", "/lib", "/lib64", "/usr"}
		var found []string
		for _, mount := range mounts {
			if containsString(expectedMountPaths, mount.path) {
				found = append(found, mount.path)
				Expect(mount.options).To(ContainElement("ro"), fmt.Sprintf("no read only permissions for %s", mount.path))
			}
		}

		Expect(found).To(ConsistOf(expectedMountPaths))
	})

	It("doesn't allow processes to read data from masked paths", func() {
		resp, err := client.Get(fmt.Sprintf("%s/masked-paths", *agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())

		Expect(string(body)).To(ContainSubstring("0+0 records in"))
		Expect(string(body)).To(ContainSubstring("0+0 records out"))
	})

	It("only has access to store, data, jobs, sys, and packages in /var/vcap", func() {
		resp, err := client.Get(fmt.Sprintf("%s/var-vcap", *agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		directories := strings.Split(strings.Trim(string(body), "\n"), "\n")
		Expect(directories).To(ConsistOf("store", "data", "jobs", "packages", "sys"))
	})

	It("only has access to its own data directory", func() {
		resp, err := client.Get(fmt.Sprintf("%s/var-vcap-data", *agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		directories := strings.Split(strings.Trim(string(body), "\n"), "\n")
		Expect(directories).To(ConsistOf("test-server", "packages"))
	})

	It("only has access to its own job directory", func() {
		resp, err := client.Get(fmt.Sprintf("%s/var-vcap-jobs", *agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		directories := strings.Split(strings.Trim(string(body), "\n"), "\n")
		Expect(directories).To(ConsistOf("test-server"))
	})

	It("is contained in a pid namespace", func() {
		resp, err := client.Get(fmt.Sprintf("%s/processes", *agentURI))
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		processes := strings.Split(strings.Trim(string(body), "\n"), "\n")

		// We expect the test agent to be the only process with the root PID
		Expect(len(processes)).To(BeNumerically(">=", 1))
		Expect(processes).To(ContainElement(MatchRegexp("1 /var/vcap/packages/test-server/bin/test-server.*")))
	})

	It("has the default path env variable", func() {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/env", *agentURI), strings.NewReader("PATH"))
		Expect(err).NotTo(HaveOccurred())

		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(ContainSubstring("/var/vcap/jobs/test-server/bin:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin:."))
	})

	It("has the default home env variable", func() {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/env", *agentURI), strings.NewReader("HOME"))
		Expect(err).NotTo(HaveOccurred())

		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(body)).To(ContainSubstring("/var/vcap/data/"))
	})

	Context("seccomp filters", func() {
		It("accepts an allowed syscall", func() {
			resp, err := client.Get(fmt.Sprintf("%s/syscall-allowed", *agentURI))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(ContainSubstring("Expected success occurred"))
		})

		It("does not accept a disallowed syscall", func() {
			resp, err := client.Get(fmt.Sprintf("%s/syscall-disallowed", *agentURI))
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(ContainSubstring("Expected error occurred"))
		})
	})

	It("it can communicate with the public internet", func() {
		req, err := http.NewRequest("GET", fmt.Sprintf("%s/curl", *agentURI), strings.NewReader("http://www.google.com"))
		Expect(err).NotTo(HaveOccurred())

		resp, err := client.Do(req)
		Expect(err).NotTo(HaveOccurred())
		defer resp.Body.Close()

		Expect(resp.StatusCode).To(Equal(http.StatusOK))
	})
})

func containsString(list []string, item string) bool {
	for _, s := range list {
		if s == item {
			return true
		}
	}
	return false
}

type mount struct {
	path    string
	options []string
}

func parseMounts(mountData string) []mount {
	results := []mount{}
	scanner := bufio.NewScanner(strings.NewReader(mountData))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		columns := strings.Split(line, " ")
		options := strings.Split(columns[3], ",")
		sort.Strings(options)

		results = append(results, mount{
			path:    columns[1],
			options: options,
		})
	}

	return results
}
