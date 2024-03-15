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
	"strings"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/opencontainers/runtime-spec/specs-go"
	"gopkg.in/yaml.v3"

	"bpm/config"
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
	SetDefaultEventuallyTimeout(5 * time.Second)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	Expect(os.RemoveAll(bpmTmpDir)).To(Succeed())
	gexec.CleanupBuildArtifacts()
})

func fileContents(path string) func() string {
	return func() string {
		contents, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		return string(contents)
	}
}

func fileLines(path string) func() []string {
	return func() []string {
		contents := fileContents(path)()
		return strings.Split(strings.Trim(contents, "\n"), "\n")
	}
}

func runcCommand(root string, args ...string) *exec.Cmd {
	args = append([]string{"--root", root}, args...)
	return exec.Command("runc", args...)
}

func runcState(root string, containerID string) specs.State {
	cmd := runcCommand(root, "state", containerID)
	cmd.Stderr = GinkgoWriter

	var data []byte
	var err error

	stateResponse := specs.State{}

	data, err = cmd.Output()
	if err != nil {
		// XXX: do something smart here based on the error
		//
		// Sometimes the state returns some junk error message but we don't
		// want to return an error because it'll cause the Eventually's to
		// fail. We probably need to return more information in the error here
		// so that we can tell if this error is temporary or not.
		return stateResponse
	}

	err = json.Unmarshal(data, &stateResponse)
	Expect(err).NotTo(HaveOccurred())

	return stateResponse
}

func prepareRunc(packagePath string) {
	runcPath, err := exec.LookPath("runc")
	Expect(err).NotTo(HaveOccurred())

	err = os.Link(runcPath, filepath.Join(packagePath, "runc"))
	Expect(err).NotTo(HaveOccurred())
}

func prepareTini(packagePath string) {
	tiniPath, err := exec.LookPath("tini")
	Expect(err).NotTo(HaveOccurred())

	// We need to copy this instead of linking it because it is used inside the
	// container and the destination of the link will not be mounted into the
	// mount namespace.
	err = copyFile(filepath.Join(packagePath, "tini"), tiniPath)
	Expect(err).NotTo(HaveOccurred())

	err = os.Chmod(filepath.Join(packagePath, "tini"), 0777)
	Expect(err).NotTo(HaveOccurred())
}

func setupBoshDirectories(root, job string) string {
	jobsDataDir := filepath.Join(root, "data", job)
	Expect(os.MkdirAll(jobsDataDir, 0755)).To(Succeed())

	runDir := filepath.Join(root, "sys", "run", "bpm-runc")
	Expect(os.MkdirAll(runDir, 0755)).To(Succeed())

	storeDir := filepath.Join(root, "store")
	Expect(os.MkdirAll(storeDir, 0755)).To(Succeed())

	dataPackagePath := filepath.Join(root, "data", "packages")
	Expect(os.MkdirAll(dataPackagePath, 0755)).To(Succeed())

	bpmPackagePath := filepath.Join(root, "packages", "bpm", "bin")
	Expect(os.MkdirAll(bpmPackagePath, 0755)).To(Succeed())

	prepareRunc(bpmPackagePath)
	prepareTini(bpmPackagePath)

	return runDir
}

func copyFile(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close() //nolint:errcheck
		return err
	}
	return d.Close()
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
	Expect(os.WriteFile(configPath, data, 0644)).To(Succeed())
}

func writeInvalidConfig(root, job string) {
	configDir := filepath.Join(root, "jobs", job, "config")
	Expect(os.MkdirAll(configDir, 0755)).To(Succeed())

	configPath := filepath.Join(configDir, "bpm.yml")
	Expect(os.WriteFile(configPath, []byte("{{"), 0644)).To(Succeed())
}

func startJob(root, bpmPath, j string) {
	startCommand := exec.Command(bpmPath, "start", j)
	startCommand.Env = append(startCommand.Env, fmt.Sprintf("BPM_BOSH_ROOT=%s", root))
	session, err := gexec.Start(startCommand, GinkgoWriter, GinkgoWriter)
	Expect(err).ShouldNot(HaveOccurred())

	<-session.Exited
	Expect(session).To(gexec.Exit(0))
}
