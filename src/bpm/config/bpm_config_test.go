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
	const (
		validID   = config.ContainerPrefix + "09aAzZ_,-."
		decodedID = "blob/foo_+bar"
		encodedID = "blob+F4------+foo_+FM------+bar"
	)

	Describe("ContainerID", func() {
		var bpmCfg *config.BPMConfig

		Context("when the job name and process name are the same", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig("", "foo", "foo")
			})

			It("encodes", func() {
				encoded := bpmCfg.ContainerID()
				Expect(encoded).To(Equal(config.ContainerPrefix + "foo"))
			})
		})

		Context("when the job name and process name are not the same", func() {
			BeforeEach(func() {
				bpmCfg = config.NewBPMConfig("", "foo", "bar")
			})

			It("encodes", func() {
				encoded := bpmCfg.ContainerID()
				Expect(encoded).To(Equal(config.ContainerPrefix + "foo.bar"))
			})
		})
	})

	Describe("Encode", func() {
		It("does not modify valid runc container ID chars", func() {
			encoded := config.Encode(validID)
			Expect(encoded).To(Equal(validID))
		})

		It("base32-encodes invalid runc container ID substrings delimited by `+`", func() {
			encoded := config.Encode(decodedID)
			Expect(encoded).To(Equal(encodedID))
		})
	})

	Describe("Decode", func() {
		It("does not modify valid runc container ID chars", func() {
			decoded, err := config.Decode(validID)
			Expect(err).To(BeNil())
			Expect(decoded).To(Equal(validID))
		})

		It("decodes based32-encoded substrings delimited by `+`", func() {
			decoded, err := config.Decode(encodedID)
			Expect(err).To(BeNil())
			Expect(decoded).To(Equal(decodedID))
		})

		Context("when container ID is empty", func() {
			It("returns empty string and does not error", func() {
				decoded, err := config.Decode("")
				Expect(err).To(BeNil())
				Expect(decoded).To(Equal(""))
			})
		})

		Context("when container ID has incorrect base32 padding", func() {
			It("errors", func() {
				_, err := config.Decode("+MZXW6--+")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("could not decode container ID (+MZXW6--+)"))
			})
		})
	})

	Context("roundtrip", func() {
		It("encodes a BPM container ID to a valid runc format and back", func() {
			id := "ÉìÅ[ÉÉìÇÕìÔÇµÇ≠Ç»Ç¢+=thoseWereOdd!!"
			encoded := config.Encode(id)
			Expect(encoded).To(Equal("+YOE4HLGDQVN4HCODRHB2ZQ4HYOK4HLGDSTBYPQVVYOD6FCNAYOD4FO6DQ7BKEKZ5+thoseWereOdd+EEQQ----+"))

			decoded, err := config.Decode(encoded)
			Expect(err).To(BeNil())
			Expect(decoded).To(Equal(id))
		})
	})
})
