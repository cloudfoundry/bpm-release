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
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	"github.com/opencontainers/runtime-spec/specs-go"
	uuid "github.com/satori/go.uuid"

	"bpm/config"
	"bpm/jobid"
)

var _ = Describe("pid", func() {
	var (
		command *exec.Cmd

		cfg config.JobConfig

		boshRoot    string
		containerID string
		job         string
		runcRoot    string
	)

	BeforeEach(func() {
		var err error

		job = uuid.NewV4().String()
		containerID = jobid.Encode(job)
		boshRoot, err = os.MkdirTemp(bpmTmpDir, "pid-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		logFile := filepath.Join(boshRoot, "sys", "log", job, "foo.log")
		cfg = newJobConfig(job, defaultBash(logFile))
		writeConfig(boshRoot, job, cfg)
	})

	JustBeforeEach(func() {
		command = exec.Command(bpmPath, "pid", job)
		command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
	})

	AfterEach(func() {
		err := runcCommand(runcRoot, "delete", "--force", containerID).Run()
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
		}
		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	It("returns the external pid", func() {
		startJob(boshRoot, bpmPath, job)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		<-session.Exited
		Expect(session).To(gexec.Exit(0))

		state := runcState(runcRoot, containerID)
		Expect(session.Out).Should(gbytes.Say(fmt.Sprintf("%d", state.Pid)))
	})

	Context("when the container is failed", func() {
		BeforeEach(func() {
			startJob(boshRoot, bpmPath, job)
			Eventually(func() specs.ContainerState { return runcState(runcRoot, containerID).Status }).Should(Equal(specs.StateRunning))
			Expect(runcCommand(runcRoot, "kill", containerID, "KILL").Run()).To(Succeed())
			Eventually(func() specs.ContainerState { return runcState(runcRoot, containerID).Status }).Should(Equal(specs.StateStopped))
		})

		It("returns an error", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())

			<-session.Exited
			Expect(session).To(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("Error: process is not running or could not be found"))
		})
	})

	Context("when the container does not exist", func() {
		It("returns an error", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())

			<-session.Exited
			Expect(session).To(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("Error: process is not running or could not be found"))
		})
	})

	Context("when no job name is specified", func() {
		It("exits with a non-zero exit code and prints the usage", func() {
			session, err := gexec.Start(exec.Command(bpmPath, "pid"), GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())

			<-session.Exited
			Expect(session).To(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("must specify a job"))
		})
	})
})
