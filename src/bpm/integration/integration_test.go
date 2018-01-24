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
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("bpm", func() {
	Context("when not run as root", func() {
		var unPrivilegedBPMDir string

		BeforeEach(func() {
			var err error
			unPrivilegedBPMDir, err = ioutil.TempDir("", "vcap-bpm")
			Expect(err).NotTo(HaveOccurred())

			f, err := os.Create(filepath.Join(unPrivilegedBPMDir, "bpm"))
			Expect(err).NotTo(HaveOccurred())
			defer f.Close()

			bpmFile, err := os.Open(bpmPath)
			Expect(err).NotTo(HaveOccurred())
			defer bpmFile.Close()

			_, err = io.Copy(f, bpmFile)
			Expect(err).NotTo(HaveOccurred())

			err = os.Chmod(filepath.Join(unPrivilegedBPMDir, "bpm"), 0777)
			Expect(err).NotTo(HaveOccurred())

			// 2000 and 3000 are test fixtures in the docker container
			err = chownR(unPrivilegedBPMDir, 2000, 3000)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			err := os.RemoveAll(unPrivilegedBPMDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns an error", func() {
			command := exec.Command(filepath.Join(unPrivilegedBPMDir, "bpm"))
			command.SysProcAttr = &syscall.SysProcAttr{}
			command.SysProcAttr.Credential = &syscall.Credential{Uid: 2000, Gid: 3000}

			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))

			Expect(session.Err).ShouldNot(gbytes.Say("Usage:"))
			Expect(session.Err).Should(gbytes.Say("bpm must be run as root. Please run 'sudo -i' to become the root user."))
		})
	})

	Context("when no arguments are provided", func() {
		It("exits with a non-zero exit code and prints the usage", func() {
			command := exec.Command(bpmPath)
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(1))
			Expect(session.Err).Should(gbytes.Say("Usage:"))
		})
	})
})

func chownR(path string, uid, gid int) error {
	return filepath.Walk(path, func(name string, _ os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, uid, gid)
		}
		return err
	})
}
