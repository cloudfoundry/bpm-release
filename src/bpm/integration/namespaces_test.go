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
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"

	"bpm/config"
)

var _ = Describe("start", func() {
	var (
		command *exec.Cmd

		cfg config.JobConfig

		boshRoot    string
		containerID string
		job         string
		runcRoot    string
		stderr      string
	)

	BeforeEach(func() {
		var err error

		job = uuid.NewV4().String()
		containerID = config.Encode(job)
		boshRoot, err = ioutil.TempDir(bpmTmpDir, "start-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		stderr = filepath.Join(boshRoot, "sys", "log", job, fmt.Sprintf("%s.stderr.log", job))
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

	Context("ipc", func() {
		var messageQueueId int

		BeforeEach(func() {
			ipcCmd := exec.Command("ipcmk", "-Q")
			output, err := ipcCmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred())

			parts := strings.Split(string(output), ":")
			Expect(parts).To(HaveLen(2))
			messageQueueId, err = strconv.Atoi(strings.Trim(parts[1], " \n"))
			Expect(err).NotTo(HaveOccurred())

			cfg = newJobConfig(job, messageQueueBash(messageQueueId))
		})

		AfterEach(func() {
			ipcCmd := exec.Command("ipcrm", "-q", strconv.Itoa(messageQueueId))
			output, err := ipcCmd.CombinedOutput()
			Expect(err).NotTo(HaveOccurred(), string(output))
		})

		It("it can only see message queues in its own namespace", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))

			Eventually(fileContents(stderr)).Should(
				ContainSubstring(fmt.Sprintf("ipcs: id %d not found", messageQueueId)),
			)
		})
	})
})
