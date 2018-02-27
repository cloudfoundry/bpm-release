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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	yaml "gopkg.in/yaml.v2"
)

// This needs to be different than /tmp as /tmp is bind mounted into the
// container
const bpmTmpDir = "/bpmtmp"

func TestBpm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var bpmPath string

var _ = SynchronizedBeforeSuite(func() []byte {
	bpmPath, err := gexec.Build("bpm/cmd/bpm")
	Expect(err).NotTo(HaveOccurred())

	Expect(os.MkdirAll(bpmTmpDir, 0755)).To(Succeed())

	return []byte(bpmPath)
}, func(data []byte) {
	bpmPath = string(data)
	SetDefaultEventuallyTimeout(2 * time.Second)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	Expect(os.RemoveAll(bpmTmpDir)).To(Succeed())
	gexec.CleanupBuildArtifacts()
})

func fileContents(path string) func() string {
	return func() string {
		contents, err := ioutil.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		return string(contents)
	}
}

func runcCommand(root string, args ...string) *exec.Cmd {
	args = append([]string{"--root", root}, args...)
	return exec.Command("runc", args...)
}

func runcState(root string, containerID string) specs.State {
	cmd := runcCommand(root, "state", containerID)
	cmd.Stderr = GinkgoWriter

	data, err := cmd.Output()
	Expect(err).NotTo(HaveOccurred())

	stateResponse := specs.State{}
	err = json.Unmarshal(data, &stateResponse)
	Expect(err).NotTo(HaveOccurred())

	return stateResponse
}

func setupBoshDirectories(root, job string) string {
	jobsDataDir := filepath.Join(root, "data", job)
	Expect(os.MkdirAll(jobsDataDir, 0755)).To(Succeed())

	storeDir := filepath.Join(root, "store")
	Expect(os.MkdirAll(storeDir, 0755)).To(Succeed())

	dataPackagePath := filepath.Join(root, "data", "packages")
	Expect(os.MkdirAll(dataPackagePath, 0755)).To(Succeed())

	bpmPackagePath := filepath.Join(root, "packages", "bpm", "bin")
	Expect(os.MkdirAll(bpmPackagePath, 0755)).To(Succeed())

	runcPath, err := exec.LookPath("runc")
	Expect(err).NotTo(HaveOccurred())

	err = os.Link(runcPath, filepath.Join(bpmPackagePath, "runc"))
	Expect(err).NotTo(HaveOccurred())

	runcRoot := filepath.Join(root, "data", "bpm", "runc")
	Expect(os.MkdirAll(runcRoot, 0755)).To(Succeed())

	return runcRoot
}

func newJobConfig(job, bash string) config.JobConfig {
	return config.JobConfig{
		Processes: []*config.ProcessConfig{
			{
				Name:       job,
				Executable: "/bin/bash",
				Args: []string{
					"-c",
					bash,
				},
			},
		},
	}
}

func writeConfig(root, job string, cfg config.JobConfig) {
	data, err := yaml.Marshal(&cfg)
	Expect(err).NotTo(HaveOccurred())

	configDir := filepath.Join(root, "jobs", job, "config")
	Expect(os.MkdirAll(configDir, 0755)).To(Succeed())

	configPath := filepath.Join(configDir, "bpm.yml")
	Expect(ioutil.WriteFile(configPath, data, 0644)).To(Succeed())
}

func writeInvalidConfig(root, job string) {
	configDir := filepath.Join(root, "jobs", job, "config")
	Expect(os.MkdirAll(configDir, 0755)).To(Succeed())

	configPath := filepath.Join(configDir, "bpm.yml")
	Expect(ioutil.WriteFile(configPath, []byte("{{"), 0644)).To(Succeed())
}

func startJob(root, bpmPath, j string) {
	startCommand := exec.Command(bpmPath, "start", j)
	startCommand.Env = append(startCommand.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", root))
	session, err := gexec.Start(startCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())
	Eventually(session).Should(gexec.Exit(0))
}
