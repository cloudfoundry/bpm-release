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
	"bpm/bosh"
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
	"bpm/jobid"
)

var _ = Describe("stop", func() {
	var (
		command *exec.Cmd

		cfg config.JobConfig

		boshRoot    string
		bpmLog      string
		containerID string
		job         string
		runcRoot    string
		stdout      string

		boshEnv *bosh.Env
		logFile bosh.Path
	)

	BeforeEach(func() {
		var err error

		job = uuid.NewV4().String()
		containerID = jobid.Encode(job)
		boshRoot, err = ioutil.TempDir(bpmTmpDir, "stop-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		boshEnv = bosh.NewEnv(boshRoot)

		runcRoot = setupBoshDirectories(boshRoot, job)

		stdout = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stdout.log", job))
		bpmLog = filepath.Join(boshRoot, "sys", "log", job, "bpm.log")
		logFile = boshEnv.LogDir(job).Join("foo.log")

		cfg = newJobConfig(job, defaultBash(logFile.Internal()))
	})

	JustBeforeEach(func() {
		writeConfig(boshRoot, job, cfg)

		startCommand := exec.Command(bpmPath, "start", job)
		startCommand.Env = append(startCommand.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
		session, err := gexec.Start(startCommand, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		<-session.Exited
		Expect(session).To(gexec.Exit(0))

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

	It("signals the container with a SIGTERM by default", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		<-session.Exited
		Expect(session).To(gexec.Exit(0))

		Eventually(fileContents(stdout)).Should(ContainSubstring("Received a TERM signal"))
	})

	It("removes the pid file", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		<-session.Exited

		Expect(session).To(gexec.Exit(0))

		Expect(filepath.Join(boshRoot, "sys", "run", "bpm", job, fmt.Sprintf("%s.pid", job))).ToNot(BeAnExistingFile())
	})

	It("removes the container and its corresponding process", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ToNot(HaveOccurred())
		<-session.Exited
		Expect(session).To(gexec.Exit(0))

		nonexistentContainerErr := runcCommand(runcRoot, "state", containerID).Run()
		Expect(nonexistentContainerErr).To(HaveOccurred())
	})

	It("removes the bundle directory", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		<-session.Exited

		Expect(session).To(gexec.Exit(0))

		bundlePath := filepath.Join(boshRoot, "data", "bpm", "bundles", job, job)
		_, err = os.Open(bundlePath)
		Expect(err).To(HaveOccurred())
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It("logs bpm internal logs to a consistent location", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		<-session.Exited
		Expect(session).To(gexec.Exit(0))

		Eventually(fileContents(bpmLog)).Should(ContainSubstring("bpm.stop.starting"))
		Eventually(fileContents(bpmLog)).Should(ContainSubstring("bpm.stop.complete"))
	})

	Context("when the job name is not specified", func() {
		It("exits with a non-zero exit code and prints the usage", func() {
			command = exec.Command(bpmPath, "stop")

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(1))

			Expect(session.Err).Should(gbytes.Say("must specify a job"))
		})
	})

	Context("when the job is already stopped", func() {
		JustBeforeEach(func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))
		})

		It("returns successfully", func() {
			secondCommand := exec.Command(bpmPath, "stop", job)
			secondCommand.Env = append(secondCommand.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			secondSession, err := gexec.Start(secondCommand, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-secondSession.Exited
			Expect(secondSession).To(gexec.Exit(0))

			Expect(fileContents(bpmLog)()).To(ContainSubstring("job-already-stopped"))
		})
	})

	Context("when the job-process does not exist", func() {
		BeforeEach(func() {
			bpmLog = filepath.Join(boshRoot, "sys", "log", "non-existent", "bpm.log")
		})

		It("ignores that and is successful", func() {
			command := exec.Command(bpmPath, "stop", "non-existent")
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))
			Expect(fileContents(bpmLog)()).To(ContainSubstring("job-already-stopped"))
		})
	})

	Context("when the job exists but the config cannot be parsed", func() {
		JustBeforeEach(func() {
			writeInvalidConfig(boshRoot, job)
		})

		It("exits 1 and logs", func() {
			command := exec.Command(bpmPath, "stop", job)
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(1))
			Expect(fileContents(bpmLog)()).To(ContainSubstring("failed-to-parse-config"))
		})
	})

	Context("when the job exists and the parsed config does not have a process that matches the job", func() {
		JustBeforeEach(func() {
			cfg = newJobConfig("definitely-not-job", defaultBash(logFile.Internal()))
			writeConfig(boshRoot, job, cfg)
		})

		It("exits 1 and logs", func() {
			command := exec.Command(bpmPath, "stop", job)
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(1))
			Expect(fileContents(bpmLog)()).To(ContainSubstring("process-not-defined"))
		})
	})

	Context("when the shutdown signal is SIGINT", func() {
		BeforeEach(func() {
			cfg.Processes[0].ShutdownSignal = "INT"
		})

		It("signals the container with a SIGINT", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ToNot(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))

			Eventually(fileContents(stdout)).Should(ContainSubstring("Received an INT signal"))
		})
	})
})
