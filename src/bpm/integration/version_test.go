// Copyright (C) 2018-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("version", func() {
	var (
		command        *exec.Cmd
		versionBPMPath string
	)

	BeforeEach(func() {
		versionBPMPath = bpmPath
	})

	Context("as a command", func() {
		JustBeforeEach(func() {
			command = exec.Command(versionBPMPath, "version")
		})

		It("returns the dev build version when compiled normally", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("[DEV BUILD]"))
		})

		Context("when it is compile with a version", func() {
			BeforeEach(func() {
				var err error
				versionBPMPath, err = gexec.Build("bpm/cmd/bpm", "-ldflags", "-X bpm/commands.Version=1.2.3")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the dev build version when compiled normally", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("1.2.3"))
			})
		})

		Context("when arguments are provided", func() {
			JustBeforeEach(func() {
				command = exec.Command(versionBPMPath, "version", "extra-argument")
			})

			It("returns the usage", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(1))
				Expect(session.Err).To(gbytes.Say("Usage:"))
			})
		})
	})

	Context("as a flag", func() {
		JustBeforeEach(func() {
			command = exec.Command(versionBPMPath, "--version")
		})

		It("returns the dev build version when compiled normally", func() {
			session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())
			Eventually(session).Should(gexec.Exit(0))
			Expect(session.Out).To(gbytes.Say("[DEV BUILD]"))
		})

		Context("when it is compile with a version", func() {
			BeforeEach(func() {
				var err error
				versionBPMPath, err = gexec.Build("bpm/cmd/bpm", "-ldflags", "-X bpm/commands.Version=1.2.3")
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the dev build version when compiled normally", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("1.2.3"))
			})
		})

		Context("when applied to a sub command", func() {
			JustBeforeEach(func() {
				command = exec.Command(versionBPMPath, "logs", "--version")
			})

			It("returns the dev build version when compiled normally", func() {
				session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
				Expect(err).ShouldNot(HaveOccurred())
				Eventually(session).Should(gexec.Exit(0))
				Expect(session.Out).To(gbytes.Say("[DEV BUILD]"))
			})
		})
	})
})
