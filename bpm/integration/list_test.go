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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"

	"bpm/config"
	"bpm/models"
)

var _ = Describe("list", func() {
	var (
		command *exec.Cmd

		cfg       config.JobConfig
		failedCfg config.JobConfig

		boshRoot          string
		containerID       string
		failedContainerID string
		failedJob         string
		invalidJob        string
		job               string
		runcRoot          string
		stoppedProcess    string
		unimplementedJob  string
	)

	BeforeEach(func() {
		var err error

		// This forces the ordering from runc list to be consistent.
		job = fmt.Sprintf("started-%s", uuid.NewV4().String())
		containerID = config.Encode(job)

		failedJob = fmt.Sprintf("failed-%s", uuid.NewV4().String())
		failedContainerID = config.Encode(failedJob)

		stoppedProcess = fmt.Sprintf("stopped-%s", uuid.NewV4().String())

		invalidJob = fmt.Sprintf("invalid-%s", uuid.NewV4().String())
		unimplementedJob = fmt.Sprintf("unimplemented-%s", uuid.NewV4().String())

		boshRoot, err = ioutil.TempDir(bpmTmpDir, "list-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		cfg = newJobConfig(job, alternativeBash)
		cfg.Processes = append(cfg.Processes, &config.ProcessConfig{
			Name:       stoppedProcess,
			Executable: "/bin/bash",
			Args: []string{
				"-c",
				alternativeBash,
			},
		})

		failedCfg = newJobConfig(failedJob, "exit 1")

		writeConfig(boshRoot, job, cfg)
		writeConfig(boshRoot, failedJob, failedCfg)
		writeInvalidConfig(boshRoot, invalidJob)
		Expect(os.MkdirAll(filepath.Join(boshRoot, "jobs", unimplementedJob), 0755)).To(Succeed())

		command = exec.Command(bpmPath, "list")
		command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
	})

	AfterEach(func() {
		err := runcCommand(runcRoot, "delete", "--force", containerID).Run()
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
		}

		err = runcCommand(runcRoot, "delete", "--force", failedContainerID).Run()
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
		}
		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	It("lists the running jobs and their state", func() {
		startJob(boshRoot, bpmPath, job)
		startJob(boshRoot, bpmPath, failedJob)

		Eventually(func() string { return runcState(runcRoot, containerID).Status }).Should(Equal("running"))
		Eventually(func() string { return runcState(runcRoot, failedContainerID).Status }).Should(Equal("stopped"))

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())

		state := runcState(runcRoot, containerID)

		Eventually(session).Should(gexec.Exit(0))
		Expect(session.Out).To(gbytes.Say("Name\\s+Pid\\s+Status"))
		Expect(session.Out).To(gbytes.Say(fmt.Sprintf("%s\\s+%s\\s+%s", failedJob, "-", models.ProcessStateFailed)))
		Expect(session.Out).To(gbytes.Say(fmt.Sprintf("%s\\s+%d\\s+%s", job, state.Pid, state.Status)))
		Expect(session.Out).To(gbytes.Say(fmt.Sprintf("%s\\s+%s\\s+%s", stoppedProcess, "-", models.ProcessStateStopped)))
		Expect(session.Err).To(gbytes.Say(fmt.Sprintf("invalid config for %s:", invalidJob)))
		Expect(session.Out).NotTo(gbytes.Say(unimplementedJob))
		Expect(session.Err).NotTo(gbytes.Say(unimplementedJob))
	})
})
