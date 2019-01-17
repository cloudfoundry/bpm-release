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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bpm/config"
	"bpm/jobid"
)

var _ = Describe("Config", func() {
	Describe("Encoding", func() {
		Context("ContainerID", func() {
			var bpmCfg *config.BPMConfig

			Context("when the job name and process name are the same", func() {
				BeforeEach(func() {
					bpmCfg = config.NewBPMConfig("", "foo", "foo")
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
					bpmCfg = config.NewBPMConfig("", "foo", "bar")
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
