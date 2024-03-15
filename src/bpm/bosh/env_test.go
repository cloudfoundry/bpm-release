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

package bosh_test

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"bpm/bosh"
)

var _ = Describe("Bosh Environment", func() {
	var root string

	BeforeEach(func() {
		var err error
		root, err = os.MkdirTemp("", "bosh_test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(root)
	})

	Describe("environment", func() {
		Context("when `root` is empty", func() {
			It("the internal and external path is `/var/vcap`", func() {
				env := bosh.NewEnv("")
				Expect(env.Root().External()).To(Equal(bosh.DefaultRoot))
				Expect(env.Root().Internal()).To(Equal(bosh.DefaultRoot))
			})
		})

		Context("when `root` is not empty", func() {
			It("returns the specified value for the external path", func() {
				env := bosh.NewEnv("/some/path")
				Expect(env.Root().External()).To(Equal("/some/path"))
			})

			It("returns the default value for the internal path", func() {
				env := bosh.NewEnv("/some/path")
				Expect(env.Root().Internal()).To(Equal(bosh.DefaultRoot))
			})
		})

		Describe("a job's data directory", func() {
			It("is a path which can be scoped to a different root", func() {
				env := bosh.NewEnv(root)
				data := env.DataDir("job_name")
				Expect(data.Internal()).To(Equal("/var/vcap/data/job_name"))
				Expect(data.External()).To(Equal(root + "/data/job_name"))
			})
		})

		Describe("a job's store directory", func() {
			It("is a path which can be scoped to a different root", func() {
				env := bosh.NewEnv(root)
				store := env.StoreDir("job_name")
				Expect(store.Internal()).To(Equal("/var/vcap/store/job_name"))
				Expect(store.External()).To(Equal(root + "/store/job_name"))
			})
		})

		Describe("a job's job directory", func() {
			It("is a path which can be scoped to a different root", func() {
				env := bosh.NewEnv(root)
				job := env.JobDir("job_name")
				Expect(job.Internal()).To(Equal("/var/vcap/jobs/job_name"))
				Expect(job.External()).To(Equal(root + "/jobs/job_name"))
			})
		})

		Describe("a job's run directory", func() {
			It("is a path which can be scoped to a different root", func() {
				env := bosh.NewEnv(root)
				run := env.RunDir("job_name")
				Expect(run.Internal()).To(Equal("/var/vcap/sys/run/job_name"))
				Expect(run.External()).To(Equal(root + "/sys/run/job_name"))
			})
		})

		Describe("a job's log directory", func() {
			It("is a path which can be scoped to a different root", func() {
				env := bosh.NewEnv(root)
				log := env.LogDir("job_name")
				Expect(log.Internal()).To(Equal("/var/vcap/sys/log/job_name"))
				Expect(log.External()).To(Equal(root + "/sys/log/job_name"))
			})
		})

		Describe("a global packages directory", func() {
			It("is a path which can be scoped to a different root", func() {
				env := bosh.NewEnv(root)
				packages := env.PackageDir()
				Expect(packages.Internal()).To(Equal("/var/vcap/packages"))
				Expect(packages.External()).To(Equal(root + "/packages"))
			})
		})

		Describe("a global data packages directory", func() {
			It("is a path which can be scoped to a different root", func() {
				env := bosh.NewEnv(root)
				packages := env.DataPackageDir()
				Expect(packages.Internal()).To(Equal("/var/vcap/data/packages"))
				Expect(packages.External()).To(Equal(root + "/data/packages"))
			})
		})
	})

	Describe("JobNames", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(root, "jobs", "job-a"), 0700)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(root, "jobs", "job-b"), 0700)).To(Succeed())
		})

		It("returns a list of BOSH job directories", func() {
			paths := bosh.NewEnv(root).JobNames()
			Expect(paths).To(ConsistOf("job-a", "job-b"))
		})
	})
})
