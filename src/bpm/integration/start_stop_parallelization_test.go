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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	uuid "github.com/satori/go.uuid"
)

var _ = Describe("start / stop parallelization", func() {
	var (
		cfg config.JobConfig

		boshRoot    string
		containerID string
		job         string
		runcRoot    string
	)

	BeforeEach(func() {
		var err error

		job = uuid.NewV4().String()
		containerID = config.Encode(job)
		boshRoot, err = ioutil.TempDir(bpmTmpDir, "start-stop-parallelization-test")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chmod(boshRoot, 0755)).To(Succeed())
		runcRoot = setupBoshDirectories(boshRoot, job)

		cfg = newJobConfig(job, waitForSigUSR1Bash)
	})

	JustBeforeEach(func() {
		writeConfig(boshRoot, job, cfg)
		startJob(boshRoot, bpmPath, job)
	})

	AfterEach(func() {
		err := runcCommand(runcRoot, "delete", "--force", containerID).Run()
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "WARNING: Failed to cleanup container: %s\n", err.Error())
		}
		Expect(os.RemoveAll(boshRoot)).To(Succeed())
	})

	It("serializes calls to start and stop", func() {
		stopCmd := exec.Command(bpmPath, "stop", job)
		stopCmd.Env = append(stopCmd.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
		stopSesh, err := gexec.Start(stopCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Consistently(stopSesh).ShouldNot(gexec.Exit())

		startCmd := exec.Command(bpmPath, "start", job)
		startCmd.Env = append(startCmd.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", boshRoot))
		startSesh, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())
		Consistently(startSesh).ShouldNot(gexec.Exit())

		Expect(runcCommand(runcRoot, "kill", containerID, "USR1").Run()).To(Succeed())

		Eventually(stopSesh).Should(gexec.Exit(0))
		Eventually(startSesh).Should(gexec.Exit(0))

		Eventually(func() string { return runcState(runcRoot, containerID).Status }).Should(Equal("running"))
	})
})
