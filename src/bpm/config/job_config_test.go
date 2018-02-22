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

package config_test

import (
	"bpm/config"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	Describe("ParseJobConfig", func() {
		var configPath string

		BeforeEach(func() {
			configPath = "fixtures/example.yml"
		})

		It("parses a yaml file into a bpm config", func() {
			cfg, err := config.ParseJobConfig(configPath)
			Expect(err).NotTo(HaveOccurred())

			expectedMemoryLimit := "100G"
			expectedOpenFilesLimit := uint64(100)

			Expect(cfg.Processes).To(HaveLen(3))

			Expect(cfg.Processes[0].Name).To(Equal("first-process"))
			Expect(cfg.Processes[0].Executable).To(Equal("/var/vcap/packages/program/bin/program-server"))
			Expect(cfg.Processes[0].Args).To(ConsistOf("--port=2424", "--host=\"localhost\""))
			Expect(cfg.Processes[0].Env).To(HaveKeyWithValue("FOO", "BAR"))
			Expect(cfg.Processes[0].Env).To(HaveKeyWithValue("BAZ", "BUZZ"))
			Expect(cfg.Processes[0].Limits.Memory).To(Equal(&expectedMemoryLimit))
			Expect(cfg.Processes[0].Limits.OpenFiles).To(Equal(&expectedOpenFilesLimit))
			Expect(cfg.Processes[0].AdditionalVolumes).To(ConsistOf(
				config.Volume{Path: "/var/vcap/data/program/foobar", Writable: true},
				config.Volume{Path: "/var/vcap/data/alternate-program"},
				config.Volume{Path: "/var/vcap/data/jna-tmp", Writable: true, AllowExecutions: true},
			))
			Expect(cfg.Processes[0].Hooks.PreStart).To(Equal("/var/vcap/jobs/program/bin/pre"))
			Expect(cfg.Processes[0].Capabilities).To(ConsistOf("NET_BIND_SERVICE"))
			Expect(cfg.Processes[0].WorkDir).To(Equal("/I/AM/A/WORKDIR"))
			Expect(cfg.Processes[0].PersistentDisk).To(BeTrue())
			Expect(cfg.Processes[0].EphemeralDisk).To(BeTrue())

			Expect(cfg.Processes[1].Name).To(Equal("second-process"))
			Expect(cfg.Processes[1].Executable).To(Equal("/I/AM/A/SECOND-EXECUTABLE"))
			Expect(cfg.Processes[1].Hooks).To(BeNil())

			Expect(cfg.Processes[2].Name).To(Equal("third-process"))
			Expect(cfg.Processes[2].Executable).To(Equal("/I/AM/A/THIRD-EXECUTABLE"))
			Expect(cfg.Processes[2].Hooks.PreStart).To(BeEmpty())
		})

		Context("when reading the file fails", func() {
			BeforeEach(func() {
				configPath = "does-not-exist"
			})

			It("returns an error", func() {
				_, err := config.ParseJobConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the yaml is invalid", func() {
			BeforeEach(func() {
				configPath = "fixtures/example-invalid-yaml.yml"
			})

			It("returns an error", func() {
				_, err := config.ParseJobConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the configuration is not valid", func() {
			BeforeEach(func() {
				configPath = "fixtures/example-invalid-config.yml"
			})

			It("returns an error", func() {
				_, err := config.ParseJobConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Validate", func() {
		var cfg *config.JobConfig

		BeforeEach(func() {
			cfg = &config.JobConfig{
				Processes: []*config.ProcessConfig{
					{
						Name:       "example",
						Executable: "executable",
						AdditionalVolumes: []config.Volume{
							{Path: "/var/vcap/data/program", Writable: true},
							{Path: "/var/vcap/store/program", Writable: false},
						},
					},
				},
			}
		})

		It("does not error on a valid config", func() {
			Expect(cfg.Validate()).To(Succeed())
		})

		Context("when the config has additional_volumes that are not nested in `/var/vcap`", func() {
			It("returns a validation error", func() {
				cfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/valid"},
					{Path: "/bin"},
				}
				Expect(cfg.Validate()).To(HaveOccurred())

				cfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/valid"},
					{Path: "/var/vcap/invalid"},
				}
				Expect(cfg.Validate()).To(HaveOccurred())

				cfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/valid"},
					{Path: "/var/vcap/data"},
				}
				Expect(cfg.Validate()).To(HaveOccurred())

				cfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/store"},
				}
				Expect(cfg.Validate()).To(HaveOccurred())

				cfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "//var/vcap/data/valid"},
				}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})

		Context("when the capabilities are not permitted", func() {
			BeforeEach(func() {
				cfg.Processes[0].Capabilities = []string{"YO_DOG_CAP"}
			})

			It("returns an error", func() {
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})

		Context("when the process does not have a name", func() {
			It("returns an error", func() {
				cfg.Processes[0].Name = ""
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})

		Context("when the config does not have an Executable", func() {
			BeforeEach(func() {
				cfg.Processes[0].Executable = ""
			})

			It("returns an error", func() {
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})
	})
})
