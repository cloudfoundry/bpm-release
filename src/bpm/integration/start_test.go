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
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	uuid "github.com/satori/go.uuid"

	"bpm/bosh"
	"bpm/config"
	"bpm/jobid"
)

var _ = Describe("start", func() {
	var (
		command *exec.Cmd

		cfg config.JobConfig

		boshRoot    string
		bpmLog      string
		containerID string
		job         string
		pidFile     string
		runcRoot    string
		stderr      string
		stdout      string

		boshEnv *bosh.Env
		logFile bosh.Path
	)

	BeforeEach(func() {
		var err error

		job = uuid.NewV4().String()
		containerID = jobid.Encode(job)
		boshRoot, err = os.MkdirTemp(bpmTmpDir, "start-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		boshEnv = bosh.NewEnv(boshRoot)

		runcRoot = setupBoshDirectories(boshRoot, job)

		stdout = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stdout.log", job))
		stderr = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stderr.log", job))
		bpmLog = filepath.Join(boshRoot, "sys", "log", job, "bpm.log")
		pidFile = filepath.Join(boshRoot, "sys", "run", "bpm", job, fmt.Sprintf("%s.pid", job))
		logFile = boshEnv.LogDir(job).Join("foo.log")

		cfg = newJobConfig(job, defaultBash(logFile.Internal()))
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
		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	It("writes the processes pid to the pidfile", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		<-session.Exited

		Expect(session).To(gexec.Exit(0))

		state := runcState(runcRoot, containerID)
		Expect(state.Status).To(Equal(specs.StateRunning))
		pidText, err := os.ReadFile(pidFile)
		Expect(err).NotTo(HaveOccurred())

		pid, err := strconv.Atoi(string(pidText))
		Expect(err).NotTo(HaveOccurred())
		Expect(pid).To(Equal(state.Pid))
	})

	It("sets the LANG environment variable", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		<-session.Exited

		Expect(session).To(gexec.Exit(0))

		Eventually(stdout).Should(BeAnExistingFile())
		Eventually(fileContents(stdout)).Should(ContainSubstring("en_US.UTF-8\n"))
	})

	It("redirects the process's stdout and stderr to their corresponding log files", func() {
		Expect(stdout).NotTo(BeAnExistingFile())
		Expect(stderr).NotTo(BeAnExistingFile())

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		<-session.Exited

		Expect(session).To(gexec.Exit(0))

		Eventually(fileContents(stdout)).Should(ContainSubstring("Logging to STDOUT"))
		Eventually(fileContents(stderr)).Should(ContainSubstring("Logging to STDERR"))
	})

	It("exposes the internal log directory for writing", func() {
		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		<-session.Exited

		Expect(session).To(gexec.Exit(0))

		Eventually(logFile.External()).Should(BeAnExistingFile())
		Eventually(fileContents(logFile.External())).Should(ContainSubstring("Logging to FILE"))
	})

	It("logs bpm internal logs to a consistent location", func() {
		Expect(bpmLog).NotTo(BeAnExistingFile())

		session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		<-session.Exited

		Expect(session).To(gexec.Exit(0))

		Eventually(fileContents(bpmLog)).Should(ContainSubstring("bpm.start.starting"))
		Eventually(fileContents(bpmLog)).Should(ContainSubstring("bpm.start.complete"))
	})

	Context("when a process name is specified", func() {
		var process string

		BeforeEach(func() {
			process = uuid.NewV4().String()
			containerID = jobid.Encode(fmt.Sprintf("%s.%s", job, process))

			cfg.Processes = append(cfg.Processes, &config.ProcessConfig{
				Name:       process,
				Executable: "/bin/bash",
				Args: []string{
					"-c",
					alternativeBash,
				},
			})

			stdout = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stdout.log", process))
			stderr = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stderr.log", process))
			pidFile = filepath.Join(boshRoot, "sys", "run", "bpm", job, fmt.Sprintf("%s.pid", process))
		})

		JustBeforeEach(func() {
			command = exec.Command(bpmPath, "start", job, "-p", process)
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
		})

		It("runs the process specified in the config", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))

			state := runcState(runcRoot, containerID)
			Expect(state.Status).To(Equal(specs.StateRunning))
			pidText, err := os.ReadFile(pidFile)
			Expect(err).NotTo(HaveOccurred())

			pid, err := strconv.Atoi(string(pidText))
			Expect(err).NotTo(HaveOccurred())
			Expect(pid).To(Equal(state.Pid))

			Eventually(fileContents(stdout)).Should(ContainSubstring("Alternate Logging to STDOUT"))
			Eventually(fileContents(stderr)).Should(ContainSubstring("Alternate Logging to STDERR"))
		})
	})

	Context("when a pre_start hook is specified", func() {
		BeforeEach(func() {
			preStart := filepath.Join(boshRoot, "pre-start")
			f, err := os.OpenFile(preStart, os.O_CREATE|os.O_RDWR, 0777)
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(preStartBash))
			Expect(err).NotTo(HaveOccurred())
			Expect(f.Close()).To(Succeed())

			cfg.Processes[0].Hooks = &config.Hooks{
				PreStart: preStart,
			}
		})

		It("executs the pre-start prior to starting the process", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))
			Eventually(fileContents(stdout)).Should(ContainSubstring("Executing Pre Start\nen_US.UTF-8\n"))
		})
	})

	Context("when persistent storage is request", func() {
		var dataFile bosh.Path

		BeforeEach(func() {
			dataFile = boshEnv.StoreDir(job).Join("data.txt")

			cfg = newJobConfig(job, defaultBash(dataFile.Internal()))
			cfg.Processes[0].PersistentDisk = true
		})

		It("exposes the storage directory as a writeable mount point", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))

			Eventually(dataFile.External()).Should(BeAnExistingFile())
			Eventually(fileContents(dataFile.External())).Should(ContainSubstring("Logging to FILE"))
		})
	})

	Context("when the bpm configuration file does not exist", func() {
		JustBeforeEach(func() {
			cfgPath := filepath.Join(boshRoot, "jobs", job, "config", "bpm.yml")
			Expect(os.RemoveAll(cfgPath)).To(Succeed())
		})

		It("exits with a non-zero exit code and prints an error", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("bpm.yml"))
		})
	})

	Context("when no job name is specified", func() {
		It("exits with a non-zero exit code and prints the usage", func() {
			session, err := gexec.Start(exec.Command(bpmPath, "start"), GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("must specify a job"))
		})
	})

	Context("when a running container exist with the same name", func() {
		var existingPid int

		JustBeforeEach(func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))

			state := runcState(runcRoot, containerID)
			Expect(state.Status).To(Equal(specs.StateRunning))
			existingPid = state.Pid
		})

		It("should not restart the container and logs", func() {
			command = exec.Command(bpmPath, "start", job)
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))
			Expect(fileContents(bpmLog)()).To(ContainSubstring("process-already-running"))

			state := runcState(runcRoot, containerID)
			Expect(state.Pid).To(Equal(existingPid))
		})
	})

	Context("when a broken runc configuration is left on the system", func() {
		JustBeforeEach(func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))

			session, err = gexec.Start(runcCommand(runcRoot, "kill", containerID), GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))
			Eventually(func() specs.ContainerState { return runcState(runcRoot, containerID).Status }, 20*time.Second).Should(Equal(specs.StateStopped))

			err = os.WriteFile(filepath.Join(runcRoot, containerID, "state.json"), nil, 0600)
			Expect(err).ToNot(HaveOccurred())
		})

		It("`bpm start` cleans up the broken-ness and starts it", func() {
			command = exec.Command(bpmPath, "start", job)
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))

			state := runcState(runcRoot, containerID)
			Expect(state.Status).To(Equal(specs.StateRunning))
		})
	})

	Context("when a stopped container exists with the same name", func() {
		JustBeforeEach(func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))

			session, err = gexec.Start(runcCommand(runcRoot, "kill", containerID), GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))
			Eventually(func() specs.ContainerState { return runcState(runcRoot, containerID).Status }).Should(Equal(specs.StateStopped))
		})

		It("`bpm start` cleans up the associated container and artifacts and starts it", func() {
			command = exec.Command(bpmPath, "start", job)
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))

			state := runcState(runcRoot, containerID)
			Expect(state.Status).To(Equal(specs.StateRunning))
		})

		Context("and the pid file does not exist", func() {
			JustBeforeEach(func() {
				err := os.RemoveAll(filepath.Join(boshRoot, "sys", "run", "bpm", job, fmt.Sprintf("%s.pid", job)))
				Expect(err).NotTo(HaveOccurred())
			})

			It("`bpm start` cleans up the associated container and artifacts and starts it", func() {
				command = exec.Command(bpmPath, "start", job)
				command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))

				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				<-session.Exited

				Expect(session).To(gexec.Exit(0))

				state := runcState(runcRoot, containerID)
				Expect(state.Status).To(Equal(specs.StateRunning))
			})
		})
	})

	Context("when the process is not defined in the bpm config", func() {
		JustBeforeEach(func() {
			command = exec.Command(bpmPath, "start", job, "-p", "I DO NOT EXIST")
			command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
		})

		It("exits with a non zero exit code and returns an error", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say(`process "I DO NOT EXIST" not present in job configuration`))
		})
	})

	Context("when specifying volumes with a globbed path", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(boshRoot, "jobs", "job-1", "config"), 0700)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(boshRoot, "jobs", "job-2", "config"), 0700)).To(Succeed())

			firstIndicator := filepath.Join(boshRoot, "jobs", "job-1", "config", "indicators.yml")
			Expect(os.WriteFile(firstIndicator, []byte("i am the first indicator file"), 0700)).To(Succeed())

			secondIndicator := filepath.Join(boshRoot, "jobs", "job-2", "config", "indicators.yml")
			Expect(os.WriteFile(secondIndicator, []byte("i am the second indicator file"), 0700)).To(Succeed())

			secret := filepath.Join(boshRoot, "jobs", "job-1", "config", "secrets.yml")
			Expect(os.WriteFile(secret, []byte("i am the secret file"), 0700)).To(Succeed())

			cfg = newJobConfig(job, findBash(filepath.Join(boshRoot, "jobs")))
			cfg.Processes[0].Unsafe = &config.Unsafe{
				UnrestrictedVolumes: []config.Volume{
					{Path: filepath.Join(boshRoot, "jobs", "*", "config", "indicators.yml")},
				},
			}
		})

		It("evaluates the glob and only mounts those directories and files into the container", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))

			Eventually(stdout).Should(BeAnExistingFile())
			Consistently(fileContents(stdout)).ShouldNot(ContainSubstring("secrets.yml"))
		})
	})

	Context("when the volume specified is a file", func() {
		BeforeEach(func() {
			randomFile := filepath.Join(boshRoot, "data", job, "random-file")
			f, err := os.OpenFile(randomFile, os.O_CREATE|os.O_RDWR, 0777)
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte("i am a random file"))
			Expect(err).NotTo(HaveOccurred())
			Expect(f.Close()).To(Succeed())

			cfg.Processes[0].AdditionalVolumes = []config.Volume{
				{Path: randomFile},
			}

			cfg.Processes[0].Args = []string{
				"-c",
				catBash(randomFile),
			}
		})

		It("properly bind mounts the file into the container", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))

			Eventually(stdout).Should(BeAnExistingFile())
			Eventually(fileContents(stdout)).Should(ContainSubstring("i am a random file"))
		})
	})
})
