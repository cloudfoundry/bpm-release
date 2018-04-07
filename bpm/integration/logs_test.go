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
)

var _ = Describe("logs", func() {
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
		containerID = config.Encode(job)
		boshRoot, err = ioutil.TempDir(bpmTmpDir, "logs-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		stdout = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stdout.log", job))
		stderr = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stderr.log", job))

		cfg = newJobConfig(job, logsBash)

		command = exec.Command(bpmPath, "logs", job)
	})

	JustBeforeEach(func() {
		writeConfig(boshRoot, job, cfg)
		startJob(boshRoot, bpmPath, job)

		Eventually(fileContents(stdout)).Should(ContainSubstring("Logging Line #100 to STDOUT"))
		Eventually(fileContents(stderr)).Should(ContainSubstring("Logging Line #100 to STDERR"))

		command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
		command.Env = append(command.Env, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))
	})

	AfterEach(func() {
		err := runcCommand(runcRoot, "delete", "--force", containerID).Run()
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
		}
		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	validateNLogLinesArePresent := func(output, source string, n int) {
		for i := 0; i < n; i++ {
			Expect(string(output)).To(ContainSubstring(fmt.Sprintf("Logging Line #%d to %s", 100-i, source)))
		}

		Expect(string(output)).NotTo(ContainSubstring(fmt.Sprintf("Logging Line #%d to %s", 100-n, source)))
	}

	It("prints the last 25 lines from stdout", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Eventually(session).Should(gexec.Exit(0))
		output := session.Out.Contents()
		validateNLogLinesArePresent(string(output), "STDOUT", 25)
	})

	Context("when the -f flag is specified", func() {
		BeforeEach(func() {
			command = exec.Command(bpmPath, "logs", job, "-f")
		})

		It("streams the logs until it receives a SIGINT signal", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session.Out).Should(gbytes.Say("Logging Line #100 to STDOUT\n"))
			Consistently(session).ShouldNot(gexec.Exit())
			session.Interrupt()
			Eventually(session).Should(gexec.Exit())
		})
	})

	Context("when the -n flag is specified", func() {
		BeforeEach(func() {
			command = exec.Command(bpmPath, "logs", job, "-n", "30")
		})

		It("prints the last n lines from stdout", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			output := session.Out.Contents()
			validateNLogLinesArePresent(string(output), "STDOUT", 30)
		})
	})

	Context("when the --err flag is specified", func() {
		BeforeEach(func() {
			command = exec.Command(bpmPath, "logs", job, "--err")
		})

		It("prints the last 25 lines from stderr", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			output := session.Out.Contents()
			validateNLogLinesArePresent(string(output), "STDERR", 25)
		})

		Context("when the -n flag is specified", func() {
			BeforeEach(func() {
				command = exec.Command(bpmPath, "logs", job, "--err", "-n", "30")
			})

			It("prints the last n lines from stderr", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				output := session.Out.Contents()
				validateNLogLinesArePresent(string(output), "STDERR", 30)
			})
		})
	})

	Context("when the --all flag is specified", func() {
		BeforeEach(func() {
			command = exec.Command(bpmPath, "logs", job, "--all")
		})

		It("prints the last 25 lines from stdout and stderr with file headers", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			output := session.Out.Contents()

			Expect(string(output)).To(ContainSubstring(stdout))
			Expect(string(output)).To(ContainSubstring(stderr))
			validateNLogLinesArePresent(string(output), "STDOUT", 25)
			validateNLogLinesArePresent(string(output), "STDERR", 25)
		})

		Context("when the -n flag is specified", func() {
			BeforeEach(func() {
				command = exec.Command(bpmPath, "logs", job, "--all", "-n", "30")
			})

			It("prints the last n lines from stdout and stderr", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				output := session.Out.Contents()
				validateNLogLinesArePresent(string(output), "STDOUT", 30)
				validateNLogLinesArePresent(string(output), "STDERR", 30)
			})
		})

		Context("when the -q flag is specified", func() {
			BeforeEach(func() {
				command = exec.Command(bpmPath, "logs", job, "--all", "-q")
			})

			It("does not print the file name headers", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				output := session.Out.Contents()

				Expect(string(output)).NotTo(ContainSubstring(stdout))
				Expect(string(output)).NotTo(ContainSubstring(stderr))
				validateNLogLinesArePresent(string(output), "STDOUT", 25)
				validateNLogLinesArePresent(string(output), "STDERR", 25)
			})
		})

		Context("when the --err flag is also specified", func() {
			BeforeEach(func() {
				command = exec.Command(bpmPath, "logs", job, "--all")
			})

			It("still prints only 25 lines from both stdout and stderr", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				output := session.Out.Contents()
				validateNLogLinesArePresent(string(output), "STDOUT", 25)
				validateNLogLinesArePresent(string(output), "STDERR", 25)
			})
		})
	})

	Context("when the process flag is provided", func() {
		var (
			process          string
			otherContainerID string
			otherStdout      string
			otherStderr      string
		)

		BeforeEach(func() {
			process = uuid.NewV4().String()
			otherContainerID = config.Encode(process)

			cfg.Processes = append(cfg.Processes, &config.ProcessConfig{
				Name:       process,
				Executable: "/bin/bash",
				Args: []string{
					"-c",
					alternativeLogsBash,
				},
			})

			otherStdout = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stdout.log", process))
			otherStderr = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stderr.log", process))

			command = exec.Command(bpmPath, "logs", job, "-p", process)
		})

		JustBeforeEach(func() {
			startCommand := exec.Command(bpmPath, "start", job, "-p", process)
			startCommand.Env = append(startCommand.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
			session, err := gexec.Start(startCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Eventually(fileContents(otherStdout)).Should(ContainSubstring("Logging Line #100 to ALT STDOUT"))
			Eventually(fileContents(otherStderr)).Should(ContainSubstring("Logging Line #100 to ALT STDERR"))

			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
			command.Env = append(command.Env, fmt.Sprintf("PATH=%s", os.Getenv("PATH")))
		})

		AfterEach(func() {
			err := runcCommand(runcRoot, "delete", "--force", otherContainerID).Run()
			if err != nil {
				fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
			}
			Expect(os.RemoveAll(boshRoot)).To(Succeed())
		})

		It("shows logs from the corresponding process", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			output := session.Out.Contents()
			validateNLogLinesArePresent(string(output), "ALT STDOUT", 25)
		})
	})

	Context("when the job does not exist", func() {
		BeforeEach(func() {
			command = exec.Command(bpmPath, "logs", "non-existant")
		})

		It("returns an error", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("Error: logs not found"))
		})
	})
})
