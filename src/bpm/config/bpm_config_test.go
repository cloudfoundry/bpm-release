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
	Describe("Encoding", func() {
		It("roundtrip encodes a bpm containerid to a valid runc format and back", func() {
			bpmContainerID := "ÉGÉìÉRÅ[ÉfÉBÉìÉOÇÕìÔÇµÇ≠Ç»oÇ¢.thoseWereOdd"
			encoded := config.Encode(bpmContainerID)

			Expect(encoded).To(Equal("YOEUPQ4JYOWMHCKSYOCVXQ4JM3BYSQWDRHB2ZQ4JJ7BYPQ4VYOWMHFGDQ7BLLQ4H4KE2BQ4HYK5W7Q4HYKRC45DIN5ZWKV3FOJSU6ZDE"))

			decoded, err := config.Decode(encoded)
			Expect(err).To(BeNil())
			Expect(decoded).To(Equal(bpmContainerID))
		})

		It("errors with incorrect base32 padding", func() {
			_, err := config.Decode("MZXW6--")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("illegal base32"))
		})

		Context("ContainerID", func() {
			var bpmCfg *config.BPMConfig

			Context("when the job name and process name are the same", func() {
				BeforeEach(func() {
					bpmCfg = config.NewBPMConfig("", "foo", "foo")
				})

				It("encodes", func() {
					encoded := bpmCfg.ContainerID()
					Expect(encoded).To(Equal("MZXW6---"))
				})
			})

			Context("when the job name and process name are not the same", func() {
				BeforeEach(func() {
					bpmCfg = config.NewBPMConfig("", "foo", "bar")
				})

				It("encodes", func() {
					encoded := bpmCfg.ContainerID()
					Expect(encoded).To(Equal("MZXW6LTCMFZA----"))
				})
			})
		})
	})
})
