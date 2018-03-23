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
	"syscall"

	"github.com/kr/pty"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("start", func() {
	var (
		command *exec.Cmd

		cfg config.JobConfig

		ptyF, ttyF *os.File

		boshRoot    string
		containerID string
		job         string
		runcRoot    string
	)

	BeforeEach(func() {
		var err error

		job = uuid.NewV4().String()
		containerID = config.Encode(job)
		boshRoot, err = ioutil.TempDir(bpmTmpDir, "start-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		logFile := filepath.Join(boshRoot, "sys", "log", job, "foo.log")
		cfg = newJobConfig(job, defaultBash(logFile))
		writeConfig(boshRoot, job, cfg)

		ptyF, ttyF, err = pty.Open()
		Expect(err).ShouldNot(HaveOccurred())
	})

	JustBeforeEach(func() {
		command = exec.Command(bpmPath, "shell", job)
		command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
		command.Env = append(command.Env, "TERM=xterm-256color")

		command.Stdin = ttyF
		command.Stdout = ttyF
		command.Stderr = ttyF
		command.SysProcAttr = &syscall.SysProcAttr{Setctty: true, Setsid: true}
	})

	AfterEach(func() {
		Expect(ptyF.Close()).To(Succeed())

		err := runcCommand(runcRoot, "delete", "--force", containerID).Run()
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
		}
		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	It("attaches to a shell running inside the container", func() {
		startJob(boshRoot, bpmPath, job)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ttyF.Close()).NotTo(HaveOccurred())

		// Validate TERM variable is set
		_, err = ptyF.Write([]byte("/bin/echo $TERM\n"))
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session.Out).Should(gbytes.Say("xterm-256color"))

		_, err = ptyF.Write([]byte("exit\n"))
		Expect(err).ShouldNot(HaveOccurred())

		Eventually(session).Should(gexec.Exit(0))
	})

	It("does not print the usage on invalid commands", func() {
		startJob(boshRoot, bpmPath, job)

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(ttyF.Close()).NotTo(HaveOccurred())

		_, err = ptyF.Write([]byte("this is not a valid command\n"))
		Expect(err).ShouldNot(HaveOccurred())

		_, err = ptyF.Write([]byte("exit\n"))
		Expect(err).ShouldNot(HaveOccurred())

		Consistently(session.Out).ShouldNot(gbytes.Say("Usage:"))
		Consistently(session.Err).ShouldNot(gbytes.Say("Usage:"))
	})

	Context("when the container does not exist", func() {
		It("returns an error", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("process is not running or could not be found"))
		})
	})

	Context("when no job name is specified", func() {
		It("exits with a non-zero exit code and prints the usage", func() {
			session, err := gexec.Start(exec.Command(bpmPath, "shell"), GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("must specify a job"))
		})
	})
})
