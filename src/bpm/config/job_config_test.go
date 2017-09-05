// Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
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

			Expect(cfg.Processes).To(HaveLen(2))

			processCfg, ok := cfg.Processes["example"]
			Expect(ok).To(BeTrue())
			Expect(processCfg.Executable).To(Equal("/var/vcap/packages/program/bin/program-server"))
			Expect(processCfg.Args).To(ConsistOf("--port=2424", "--host=\"localhost\""))
			Expect(processCfg.Env).To(ConsistOf("FOO=BAR", "BAZ=BUZZ"))
			Expect(processCfg.Limits.Memory).To(Equal(&expectedMemoryLimit))
			Expect(processCfg.Limits.OpenFiles).To(Equal(&expectedOpenFilesLimit))
			Expect(processCfg.Volumes).To(ConsistOf("/var/vcap/data/program/foobar", "/var/vcap/data/alternate-program"))
			Expect(processCfg.Hooks.PreStart).To(Equal("/var/vcap/jobs/program/bin/pre"))

			alternateConfig, ok := cfg.Processes["alternate-example"]
			Expect(ok).To(BeTrue())
			Expect(alternateConfig.Executable).To(Equal("/I/AM/AN/EXECUTABLE"))
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
				configPath = "fixtures/example-invalid.yml"
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
				Processes: map[string]*config.ProcessConfig{
					"example": &config.ProcessConfig{
						Executable: "executable",
						Volumes: []string{
							"/var/vcap/data/program",
							"/var/vcap/store/program",
						},
					},
				},
			}
		})

		It("does not error on a valid config", func() {
			Expect(cfg.Validate()).To(Succeed())
		})

		Context("when the config has volumes that are not nested in `/var/vcap`", func() {
			It("returns a validation error", func() {
				cfg.Processes["example"].Volumes = []string{"/var/vcap/data/valid", "/bin"}
				Expect(cfg.Validate()).To(HaveOccurred())

				cfg.Processes["example"].Volumes = []string{"/var/vcap/data/valid", "/var/vcap/invalid"}
				Expect(cfg.Validate()).To(HaveOccurred())

				cfg.Processes["example"].Volumes = []string{"/var/vcap/data/valid", "/var/vcap/data"}
				Expect(cfg.Validate()).To(HaveOccurred())

				cfg.Processes["example"].Volumes = []string{"/var/vcap/store"}
				Expect(cfg.Validate()).To(HaveOccurred())

				cfg.Processes["example"].Volumes = []string{"//var/vcap/data/valid"}
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})

		Context("when the config does not have an Executable", func() {
			BeforeEach(func() {
				cfg.Processes["example"].Executable = ""
			})

			It("returns an error", func() {
				Expect(cfg.Validate()).To(HaveOccurred())
			})
		})
	})
})
