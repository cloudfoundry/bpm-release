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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bpm/bosh"
	"bpm/config"
)

var _ = Describe("Config", func() {
	var boshEnv *bosh.Env

	BeforeEach(func() {
		boshEnv = bosh.NewEnv("")
	})

	Describe("ParseJobConfig", func() {
		var configPath string

		BeforeEach(func() {
			configPath = "testdata/example.yml"
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
				config.Volume{Path: "/var/vcap/data/shared", Shared: true},
			))
			Expect(cfg.Processes[0].Hooks.PreStart).To(Equal("/var/vcap/jobs/program/bin/pre"))
			Expect(cfg.Processes[0].Capabilities).To(ConsistOf("NET_BIND_SERVICE", "SYS_TIME"))
			Expect(cfg.Processes[0].WorkDir).To(Equal("/I/AM/A/WORKDIR"))
			Expect(cfg.Processes[0].PersistentDisk).To(BeTrue())
			Expect(cfg.Processes[0].EphemeralDisk).To(BeTrue())
			Expect(cfg.Processes[0].Unsafe.Privileged).To(BeTrue())
			Expect(cfg.Processes[0].Unsafe.HostPidNamespace).To(BeTrue())
			Expect(cfg.Processes[0].Unsafe.UnrestrictedVolumes).To(ConsistOf(
				config.Volume{Path: "/", Writable: true},
				config.Volume{Path: "/etc"},
				config.Volume{Path: "/foobar", Writable: true, AllowExecutions: true},
			))

			Expect(cfg.Processes[1].Name).To(Equal("second-process"))
			Expect(cfg.Processes[1].Executable).To(Equal("/I/AM/A/SECOND-EXECUTABLE"))
			Expect(cfg.Processes[1].Hooks).To(BeNil())
			Expect(cfg.Processes[1].Unsafe).To(BeNil())

			Expect(cfg.Processes[2].Name).To(Equal("third-process"))
			Expect(cfg.Processes[2].Executable).To(Equal("/I/AM/A/THIRD-EXECUTABLE"))
			Expect(cfg.Processes[2].Hooks.PreStart).To(BeEmpty())
			Expect(cfg.Processes[2].Unsafe).To(BeNil())
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
				configPath = "testdata/example-invalid-yaml.yml"
			})

			It("returns an error", func() {
				_, err := config.ParseJobConfig(configPath)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Validate", func() {
		var jobCfg *config.JobConfig

		BeforeEach(func() {
			jobCfg = &config.JobConfig{
				Processes: []*config.ProcessConfig{
					{
						Name:       "example",
						Executable: "executable",
						AdditionalVolumes: []config.Volume{
							{Path: "/var/vcap/data/valid"},
							{Path: "/var/vcap/store/valid"},
							{Path: "/var/vcap/sys/run/valid"},
						},
					},
				},
			}
		})

		It("does not error on a valid config", func() {
			Expect(jobCfg.Validate(boshEnv, []string{})).To(Succeed())
		})

		Context("when the config has additional_volumes that are not nested in the bosh root", func() {
			It("returns a validation error", func() {
				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/valid"},
					{Path: "/bin"},
				}
				Expect(jobCfg.Validate(boshEnv, []string{})).To(HaveOccurred())

				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/valid"},
					{Path: "/var/invalid"},
				}
				Expect(jobCfg.Validate(boshEnv, []string{})).To(HaveOccurred())

				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/valid"},
					{Path: "/var/vcap"},
				}
				Expect(jobCfg.Validate(boshEnv, []string{})).To(HaveOccurred())

				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "//var/vcap/data/valid"},
				}
				Expect(jobCfg.Validate(boshEnv, []string{})).To(HaveOccurred())
			})
		})

		Context("when the config has additional_volumes that conflict with default volumes", func() {
			It("returns a validation error", func() {
				jobCfg.Processes[0].AdditionalVolumes = []config.Volume{
					{Path: "/var/vcap/data/job-name"},
				}
				Expect(jobCfg.Validate(boshEnv, []string{
					"/var/vcap/data/job-name",
				})).To(HaveOccurred())
			})
		})

		Context("when the process does not have a name", func() {
			It("returns an error", func() {
				jobCfg.Processes[0].Name = ""
				Expect(jobCfg.Validate(boshEnv, []string{})).To(HaveOccurred())
			})
		})

		Context("when the config does not have an Executable", func() {
			BeforeEach(func() {
				jobCfg.Processes[0].Executable = ""
			})

			It("returns an error", func() {
				Expect(jobCfg.Validate(boshEnv, []string{})).To(HaveOccurred())
			})
		})
	})

	Describe("AddVolumes", func() {
		var cfg *config.ProcessConfig

		BeforeEach(func() {
			cfg = &config.ProcessConfig{
				Name:       "name",
				Executable: "executable",
			}
		})

		It("adds the volumes to the list of AdditionalVolumes", func() {
			err := cfg.AddVolumes(
				[]string{
					"/var/vcap/data/volume1",
					"/var/vcap/store/volume2:writable",
					"/var/vcap/data/volume3:mount_only",
					"/var/vcap/store/volume4:allow_executions",
					"/var/vcap/data/volume5:writable,mount_only,allow_executions",
					"/var/vcap/data/volume6:shared",
				},
				boshEnv,
				[]string{},
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(cfg.AdditionalVolumes).To(HaveLen(6))
			Expect(cfg.AdditionalVolumes).To(ContainElement(config.Volume{
				Path: "/var/vcap/data/volume1",
			}))
			Expect(cfg.AdditionalVolumes).To(ContainElement(config.Volume{
				Path:     "/var/vcap/store/volume2",
				Writable: true,
			}))
			Expect(cfg.AdditionalVolumes).To(ContainElement(config.Volume{
				Path:      "/var/vcap/data/volume3",
				MountOnly: true,
			}))
			Expect(cfg.AdditionalVolumes).To(ContainElement(config.Volume{
				Path:            "/var/vcap/store/volume4",
				AllowExecutions: true,
			}))
			Expect(cfg.AdditionalVolumes).To(ContainElement(config.Volume{
				Path:            "/var/vcap/data/volume5",
				Writable:        true,
				MountOnly:       true,
				AllowExecutions: true,
			}))
			Expect(cfg.AdditionalVolumes).To(ContainElement(config.Volume{
				Path:   "/var/vcap/data/volume6",
				Shared: true,
			}))
		})

		Context("when the volume definition contains an invalid option", func() {
			It("returns an error", func() {
				err := cfg.AddVolumes(
					[]string{"/path/to/volume1:non_existent"},
					boshEnv,
					[]string{},
				)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the volume has too many fields", func() {
			It("returns an error", func() {
				err := cfg.AddVolumes(
					[]string{"/path/to/volume1:writable:invalid_field"},
					boshEnv,
					[]string{},
				)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the volumes are invalid", func() {
			It("returns an error", func() {
				Expect(cfg.AddVolumes(
					[]string{"/outside/volume1:writable"},
					boshEnv,
					[]string{},
				)).To(HaveOccurred())

				Expect(cfg.AddVolumes(
					[]string{"/path/to/volume1:writable"},
					boshEnv,
					[]string{},
				)).To(HaveOccurred())

				Expect(cfg.AddVolumes(
					[]string{"/path/to/data:writable"},
					boshEnv,
					[]string{"/path/to/data"},
				)).To(HaveOccurred())
			})
		})
	})

	Describe("AddEnvVars", func() {
		var cfg *config.ProcessConfig

		BeforeEach(func() {
			cfg = &config.ProcessConfig{
				Name:       "name",
				Executable: "executable",
			}
		})

		It("adds the environment variables to the Env map", func() {
			err := cfg.AddEnvVars(
				[]string{
					"SIMPLE=values",
					"KEY=will-be-overridden",
					"KEY=later-wins",
					"OTHER=handles=equals=signs",
				},
				boshEnv,
				[]string{},
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(cfg.Env).To(HaveKeyWithValue("SIMPLE", "values"))
			Expect(cfg.Env).To(HaveKeyWithValue("KEY", "later-wins"))
			Expect(cfg.Env).To(HaveKeyWithValue("OTHER", "handles=equals=signs"))
		})

		Context("when the environment definition contains an invalid option", func() {
			It("returns an error", func() {
				err := cfg.AddEnvVars(
					[]string{"INVALID"},
					boshEnv,
					[]string{},
				)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the process config has environment variables", func() {
			It("adds the environment variables to the Env map", func() {
				cfg.Env = map[string]string{"SIMPLE": "values"}

				err := cfg.AddEnvVars([]string{"ANOTHER=value"}, boshEnv, []string{})
				Expect(err).NotTo(HaveOccurred())

				Expect(cfg.Env).To(HaveKeyWithValue("SIMPLE", "values"))
				Expect(cfg.Env).To(HaveKeyWithValue("ANOTHER", "value"))
			})
		})
	})
})
