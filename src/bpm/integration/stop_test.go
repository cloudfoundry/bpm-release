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
	"bpm/config"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("stop", func() {
	var (
		command *exec.Cmd

		cfg config.JobConfig

		boshRoot    string
		bpmLog      string
		containerID string
		job         string
		logFile     string
		runcRoot    string
		stdout      string
	)

	BeforeEach(func() {
		var err error

		job = uuid.NewV4().String()
		containerID = config.Encode(job)
		boshRoot, err = ioutil.TempDir(bpmTmpDir, "stop-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		stdout = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stdout.log", job))
		bpmLog = filepath.Join(boshRoot, "sys", "log", job, "bpm.log")
		logFile = filepath.Join(boshRoot, "sys", "log", job, "foo.log")

		cfg = newJobConfig(job, defaultBash(logFile))
	})

	JustBeforeEach(func() {
		writeConfig(boshRoot, job, cfg)

		startCommand := exec.Command(bpmPath, "start", job)
		startCommand.Env = append(startCommand.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
		session, err := gexec.Start(startCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		command = exec.Command(bpmPath, "stop", job)
		command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
	})

	AfterEach(func() {
		err := runcCommand(runcRoot, "delete", "--force", containerID).Run()
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
		}
		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	It("signals the container with a SIGTERM", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		Eventually(fileContents(stdout)).Should(ContainSubstring("Received a Signal"))
	})

	It("removes the container and its corresponding process", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		Expect(runcCommand(runcRoot, "state", containerID).Run()).To(HaveOccurred())
	})

	It("removes the bundle directory", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		bundlePath := filepath.Join(boshRoot, "data", "bpm", "bundles", job, job)
		_, err = os.Open(bundlePath)
		Expect(err).To(HaveOccurred())
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("logs bpm internal logs to a consistent location", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))

		Eventually(fileContents(bpmLog)).Should(ContainSubstring("bpm.stop.starting"))
		Eventually(fileContents(bpmLog)).Should(ContainSubstring("bpm.stop.complete"))
	})

	Context("when the job name is not specified", func() {
		It("exits with a non-zero exit code and prints the usage", func() {
			command = exec.Command(bpmPath, "stop")

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).Should(gbytes.Say("must specify a job"))
		})
	})

	Context("when the job is already stopped", func() {
		JustBeforeEach(func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
		})

		It("returns successfully", func() {
			secondCommand := exec.Command(bpmPath, "stop", job)
			secondCommand.Env = append(secondCommand.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			secondSession, err := gexec.Start(secondCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(secondSession).Should(gexec.Exit(0))
			Expect(fileContents(bpmLog)()).To(ContainSubstring("job-already-stopped"))
		})
	})

	Context("when the job-process doesn't not exist", func() {
		BeforeEach(func() {
			bpmLog = filepath.Join(boshRoot, "sys", "log", "non-existant", "bpm.log")
		})

		It("ignores that and is successful", func() {
			command := exec.Command(bpmPath, "stop", "non-existant")
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(fileContents(bpmLog)()).To(ContainSubstring("job-already-stopped"))
		})
	})
})
