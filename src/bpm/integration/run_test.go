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

package integration_test

import (
	"bufio"
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"bpm/integration/sandbox"
)

var _ = Describe("BPM run command", func() {
	var (
		s         *sandbox.Sandbox
		errandJob string
	)

	BeforeEach(func() {
		s = sandbox.New(GinkgoTB())
		errandJob = uniqueJobName("errand")
	})

	AfterEach(func() {
		if s != nil {
			s.Cleanup()
		}
	})

	Context("bpm run", func() {
		const (
			stdoutSentinel = "stdout"
			stderrSentinel = "stderr"
		)

		BeforeEach(func() {
			s.LoadFixture(errandJob, "testdata/errand.yml")
		})

		It("executes bpm run, captures stdout/stderr, logs to files and cleans up pid files", func() {
			cmd := s.BPMCmd("run", errandJob, "-p", "errand")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Expect(session.Out).To(gbytes.Say(stdoutSentinel))
			Expect(session.Err).To(gbytes.Say(stderrSentinel))

			stdoutLogPath := s.Path("sys", "log", errandJob, "errand.stdout.log")
			Expect(stdoutLogPath).To(BeAnExistingFile())
			stdoutLogContents, err := os.ReadFile(stdoutLogPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stdoutLogContents)).To(ContainSubstring(stdoutSentinel))

			stderrLogPath := s.Path("sys", "log", errandJob, "errand.stderr.log")
			Expect(stderrLogPath).To(BeAnExistingFile())
			stderrLogContents, err := os.ReadFile(stderrLogPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(stderrLogContents)).To(ContainSubstring(stderrSentinel))

			pidfile := s.Path("sys", "run", "bpm", errandJob, "errand.pid")
			Expect(pidfile).NotTo(BeAnExistingFile())
		})

	})

	Context("with -e environment variable flags", func() {
		const sentinel = "sentinel"

		BeforeEach(func() {
			s.LoadFixture(errandJob, "testdata/env-flag.yml")
		})

		It("passes the environment variables to the process", func() {
			cmd := s.BPMCmd(
				"run",
				errandJob,
				"-p", "errand",
				"-e", fmt.Sprintf("ENVKEY=%s", sentinel),
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Expect(session.Out).To(gbytes.Say(sentinel))
		})
	})

	Context("with -v volume flags", func() {
		var (
			extraVolumeDir  string
			extraVolumeFile string
			sentinel        = "success"
		)

		BeforeEach(func() {
			s.LoadFixture(errandJob, "testdata/volume-flag.yml")
			extraVolumeDir = s.Path("data", "extra-volume")
			extraVolumeFile = filepath.Join(extraVolumeDir, "data.txt")
		})

		It("mounts the specified volume, verifies mount options and file content", func() {
			cmd := s.BPMCmd(
				"run",
				errandJob,
				"-p", "errand",
				"-v", fmt.Sprintf("%s:writable,allow_executions", extraVolumeDir),
				"-e", fmt.Sprintf("FILE_TO_WRITE_TO=%s", extraVolumeFile),
			)
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			mounts, err := parseFstab(session.Out.Contents())
			Expect(err).NotTo(HaveOccurred())
			Expect(mounts).To(HaveLen(1))

			firstMount := mounts[0]
			Expect(firstMount.MountPoint).To(Equal(extraVolumeDir))
			Expect(mountHasOption(firstMount, "ro")).To(BeFalse())
			Expect(mountHasOption(firstMount, "noexec")).To(BeFalse())

			Expect(extraVolumeFile).To(BeAnExistingFile())
			fileContents, err := os.ReadFile(extraVolumeFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContents)).To(ContainSubstring(sentinel))
		})
	})

	Context("when the process fails", func() {
		BeforeEach(func() {
			s.LoadFixture("oops", "testdata/failure.yml")
		})

		It("returns a non-zero exit code", func() {
			cmd := s.BPMCmd("run", "oops")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode()).ShouldNot(Equal(0))
		})
	})

	Context("when the process exits with an unusual status", func() {
		BeforeEach(func() {
			// exit status 6
			s.LoadFixture("odd", "testdata/odd-status.yml")
		})

		It("bpm exits with the same status code", func() {
			cmd := s.BPMCmd("run", "odd")
			session, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(6))
		})
	})

})

// uniqueJobName appends a random suffix to a job name.
// This is used by tests that run the same job in parallel inorder to not conflict with each other.
func uniqueJobName(prefix string) string {
	const charset = "0123456789abcdef"
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("%s-%s", prefix, string(b))
}

type mount struct {
	MountPoint string
	Options    []string
}

// ParseFstab parses byte slices which contain the contents of files formatted
// as described by fstab(5).
func parseFstab(contents []byte) ([]mount, error) {
	var mounts []mount

	r := bytes.NewBuffer(contents)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			return nil, fmt.Errorf("invalid mount: %s", scanner.Text())
		}

		options := strings.Split(fields[3], ",")
		mounts = append(mounts, mount{
			MountPoint: fields[1],
			Options:    options,
		})
	}

	return mounts, nil
}

func mountHasOption(m mount, opt string) bool {
	for _, o := range m.Options {
		if o == opt {
			return true
		}
	}

	return false
}
