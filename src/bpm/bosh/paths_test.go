// Copyright (C) 2019-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bpm/bosh"
)

var _ = Describe("Bosh Paths", func() {
	var path bosh.Path

	BeforeEach(func() {
		env := bosh.NewEnv("/other/root")
		path = env.Root()
	})

	Describe("joining paths", func() {
		It("can have single elements joined", func() {
			newPath := path.Join("element")

			Expect(newPath.Internal()).To(Equal("/var/vcap/element"))
			Expect(newPath.External()).To(Equal("/other/root/element"))
		})

		It("can have multiple elements joined", func() {
			newPath := path.Join("element", "another_element")

			Expect(newPath.Internal()).To(Equal("/var/vcap/element/another_element"))
			Expect(newPath.External()).To(Equal("/other/root/element/another_element"))
		})

		It("does not mutate the original path", func() {
			path.Join("element")

			Expect(path.Internal()).To(Equal("/var/vcap"))
		})
	})
})
