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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bpm/bosh"
	"bpm/config"
	"bpm/jobid"
)

var _ = Describe("Binary paths", func() {
	var (
		env      *bosh.Env
		boshRoot string
	)

	BeforeEach(func() {
		tmpDir, err := os.MkdirTemp("", "binary_paths_test")
		Expect(err).NotTo(HaveOccurred())
		boshRoot, err = filepath.EvalSymlinks(tmpDir)
		Expect(err).NotTo(HaveOccurred())
		env = bosh.NewEnv(boshRoot)

		originalPackageDir, packageDirSet := os.LookupEnv("BPM_PACKAGE_DIR")
		DeferCleanup(func() {
			if packageDirSet {
				Expect(os.Setenv("BPM_PACKAGE_DIR", originalPackageDir)).To(Succeed())
			} else {
				Expect(os.Unsetenv("BPM_PACKAGE_DIR")).To(Succeed())
			}
			Expect(os.RemoveAll(boshRoot)).To(Succeed())
		})
	})

	Describe("ValidatePackageDir", func() {
		It("accepts an unset BPM_PACKAGE_DIR", func() {
			Expect(os.Unsetenv("BPM_PACKAGE_DIR")).To(Succeed())
			Expect(config.ValidatePackageDir()).To(Succeed())
		})

		It("accepts an absolute BPM_PACKAGE_DIR", func() {
			Expect(os.Setenv("BPM_PACKAGE_DIR", "/usr/libexec/bpm")).To(Succeed())
			Expect(config.ValidatePackageDir()).To(Succeed())
		})

		It("rejects a relative BPM_PACKAGE_DIR", func() {
			Expect(os.Setenv("BPM_PACKAGE_DIR", "usr/libexec/bpm")).To(Succeed())
			Expect(config.ValidatePackageDir()).To(MatchError(ContainSubstring("must be an absolute path")))
		})
	})

	Describe("RuncPath", func() {
		Context("when BPM_PACKAGE_DIR is unset", func() {
			BeforeEach(func() {
				Expect(os.Unsetenv("BPM_PACKAGE_DIR")).To(Succeed())
			})

			It("returns the path in the BOSH package directory", func() {
				Expect(config.RuncPath(env)).To(Equal(filepath.Join(boshRoot, "packages", "bpm", "bin", "runc")))
			})
		})

		Context("when BPM_PACKAGE_DIR is set", func() {
			BeforeEach(func() {
				Expect(os.Setenv("BPM_PACKAGE_DIR", "/usr/libexec/bpm")).To(Succeed())
			})

			It("returns the path under BPM_PACKAGE_DIR", func() {
				Expect(config.RuncPath(env)).To(Equal("/usr/libexec/bpm/bin/runc"))
			})
		})
	})

	Describe("TiniPath", func() {
		var bpmCfg *config.BPMConfig

		BeforeEach(func() {
			bpmCfg = config.NewBPMConfig(env, "foo", "foo")
		})

		Context("when BPM_PACKAGE_DIR is unset", func() {
			BeforeEach(func() {
				Expect(os.Unsetenv("BPM_PACKAGE_DIR")).To(Succeed())
			})

			It("returns the internal path in the BOSH package directory", func() {
				Expect(bpmCfg.TiniPath()).To(Equal("/var/vcap/packages/bpm/bin/tini"))
			})
		})

		Context("when BPM_PACKAGE_DIR is set", func() {
			BeforeEach(func() {
				Expect(os.Setenv("BPM_PACKAGE_DIR", "/usr/libexec/bpm")).To(Succeed())
			})

			It("returns the path under BPM_PACKAGE_DIR", func() {
				Expect(bpmCfg.TiniPath()).To(Equal("/usr/libexec/bpm/bin/tini"))
			})
		})
	})
})

var _ = Describe("Config", func() {
	Describe("Encoding", func() {
		Context("ContainerID", func() {
			var bpmCfg *config.BPMConfig

			Context("when the job name and process name are the same", func() {
				BeforeEach(func() {
					env := bosh.NewEnv("")
					bpmCfg = config.NewBPMConfig(env, "foo", "foo")
				})

				It("encodes", func() {
					encoded := bpmCfg.ContainerID()
					decoded, err := jobid.Decode(encoded)
					Expect(err).NotTo(HaveOccurred())
					Expect(decoded).To(Equal("foo"))
				})
			})

			Context("when the job name and process name are not the same", func() {
				BeforeEach(func() {
					env := bosh.NewEnv("")
					bpmCfg = config.NewBPMConfig(env, "foo", "bar")
				})

				It("encodes", func() {
					encoded := bpmCfg.ContainerID()
					decoded, err := jobid.Decode(encoded)
					Expect(err).NotTo(HaveOccurred())
					Expect(decoded).To(Equal("foo.bar"))
				})
			})
		})
	})
})
