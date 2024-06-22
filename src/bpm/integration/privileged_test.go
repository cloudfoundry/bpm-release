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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"

	"bpm/config"
	"bpm/jobid"
)

var _ = Describe("privileged containers", func() {
	var (
		command *exec.Cmd

		cfg config.JobConfig

		boshRoot    string
		containerID string
		job         string
		runcRoot    string
		stderr      string
		stdout      string
	)

	BeforeEach(func() {
		var err error

		job = uuid.NewV4().String()
		containerID = jobid.Encode(job)
		boshRoot, err = os.MkdirTemp(bpmTmpDir, "start-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		stdout = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stdout.log", job))
		stderr = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stderr.log", job))

		cfg = newJobConfig(job, privilegedBash)

		cfg.Processes[0].Unsafe = &config.Unsafe{
			Privileged: true,
		}
	})

	JustBeforeEach(func() {
		writeConfig(boshRoot, job, cfg)
		command = exec.Command(bpmPath, "start", job)
		command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
	})

	AfterEach(func() {
		err := runcCommand(runcRoot, "delete", "--force", containerID).Run()
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
		}
		copyContentsToGinkgoWrite(stdout)
		copyContentsToGinkgoWrite(stderr)

		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	It("starts the process as the root user without seccomp and default privileges", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		<-session.Exited

		Expect(session).To(gexec.Exit(0))

		Eventually(stdout).Should(BeAnExistingFile())
		Eventually(fileContents(stdout)).Should(ContainSubstring("Running as root"))
		Eventually(fileContents(stdout)).Should(MatchRegexp("Privileges:\\s?CapEff:\\s?0000003fffffffff\\s?"))
		Eventually(fileContents(stdout)).Should(ContainSubstring("No nosuid mounts"))
	})
})
