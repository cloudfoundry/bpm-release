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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/opencontainers/runtime-spec/specs-go"
	uuid "github.com/satori/go.uuid"

	"bpm/bosh"
	"bpm/config"
	"bpm/jobid"
)

var _ = Describe("resource limits", func() {
	var (
		command *exec.Cmd

		cfg config.JobConfig

		boshRoot    string
		boshEnv     *bosh.Env
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
		boshRoot, err = os.MkdirTemp(bpmTmpDir, "resource-limits-test")
		Expect(err).NotTo(HaveOccurred())
		boshEnv = bosh.NewEnv(boshRoot)
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		stderr = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stderr.log", job))
		stdout = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stdout.log", job))
	})

	JustBeforeEach(func() {
		writeConfig(boshRoot, job, cfg)
		command = exec.Command(bpmPath, "start", job)
		command.Env = append(command.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
	})

	AfterEach(func() {
		err := runcCommand(runcRoot, "delete", "--force", containerID).Run()
		if err != nil {
			GinkgoWriter.Printf("WARNING: Failed to cleanup container: %s\n", err.Error())
		}
		copyContentsToGinkgoWrite(stderr)
		copyContentsToGinkgoWrite(stdout)

		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	Context("memory", func() {
		BeforeEach(func() {
			cfg = newJobConfig(job, memoryLeakBash)
			limit := "16M"
			cfg.Processes[0].Limits = &config.Limits{Memory: &limit}
		})

		streamOOMEvents := func(stdout io.Reader) chan event {
			oomEvents := make(chan event)
			decoder := json.NewDecoder(stdout)

			go func() {
				defer GinkgoRecover()
				defer close(oomEvents)

				for {
					var actualEvent event
					if err := decoder.Decode(&actualEvent); err != nil {
						return
					}

					if actualEvent.Type == "oom" {
						oomEvents <- actualEvent
					}
					time.Sleep(100 * time.Millisecond)
				}
			}()

			return oomEvents
		}

		It("gets OOMed when it exceeds its memory limit", func() {
			GinkgoWriter.Printf("If this test fails, then make sure you have enabled swap accounting! Details are in the README.")

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(0))
			Eventually(func() specs.ContainerState { return runcState(runcRoot, containerID).Status }).Should(Equal(specs.StateRunning))

			eventsCmd := runcCommand(runcRoot, "events", containerID)
			stdout, err := eventsCmd.StdoutPipe()
			Expect(err).NotTo(HaveOccurred())

			oomEventsChan := streamOOMEvents(stdout)
			Expect(eventsCmd.Start()).To(Succeed())

			Expect(runcCommand(runcRoot, "kill", containerID).Run()).To(Succeed())
			Eventually(oomEventsChan).Should(Receive())
			Expect(eventsCmd.Process.Kill()).To(Succeed())
			Eventually(oomEventsChan).Should(BeClosed())
		})
	})

	Context("open files", func() {
		BeforeEach(func() {
			cfg = newJobConfig(job, fileLeakBash(boshEnv.DataDir(job).Internal()))
			limit := uint64(10)
			cfg.Processes[0].Limits = &config.Limits{OpenFiles: &limit}
			cfg.Processes[0].EphemeralDisk = true
		})

		It("cannot open more files than permitted", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			<-session.Exited
			Expect(session).To(gexec.Exit(1))
			Eventually(fileContents(stderr)).Should(ContainSubstring("too many open files"))
		})
	})

	Context("processes", func() {
		BeforeEach(func() {
			cfg = newJobConfig(job, processLeakBash)
			limit := int64(50)
			cfg.Processes[0].Limits = &config.Limits{Processes: &limit}
		})

		It("cannot create more processes than permitted", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			<-session.Exited

			Expect(session).To(gexec.Exit(0))

			Eventually(func() specs.ContainerState { return runcState(runcRoot, containerID).Status }).Should(Equal(specs.StateRunning))
			Expect(runcCommand(runcRoot, "kill", containerID).Run()).To(Succeed())
			Eventually(fileContents(stderr)).Should(ContainSubstring("fork: retry: Resource temporarily unavailable"))
		})
	})
})

type event struct {
	Type      string                 `json:"type"`
	Arbitrary map[string]interface{} `json:",inline"`
}
