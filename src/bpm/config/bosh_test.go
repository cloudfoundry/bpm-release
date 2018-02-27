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

package config_test

import (
	"bpm/config"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bosh", func() {
	var root string

	BeforeEach(func() {
		var err error
		root, err = ioutil.TempDir("", "bosh_test")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(root)
	})

	Describe("NewBosh", func() {
		Context("when `root` is empty", func() {
			It("returns `/var/vcap`", func() {
				bosh := config.NewBosh("")
				Expect(bosh.Root()).To(Equal(config.DefaultBoshRoot))
			})
		})

		Context("when `root` is NOT empty", func() {
			It("returns the specified value", func() {
				bosh := config.NewBosh("some/path")
				Expect(bosh.Root()).To(Equal("some/path"))
			})
		})
	})

	Describe("JobNames", func() {
		BeforeEach(func() {
			Expect(os.MkdirAll(filepath.Join(root, "jobs", "job-a"), 0700)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(root, "jobs", "job-b"), 0700)).To(Succeed())
		})

		It("returns a list of BOSH job directories", func() {
			paths := config.NewBosh(root).JobNames()
			Expect(paths).To(ConsistOf("job-a", "job-b"))
		})
	})
})
