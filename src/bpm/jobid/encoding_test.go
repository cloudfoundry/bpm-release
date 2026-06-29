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

package jobid

import (
	"os"
	"path/filepath"
	"regexp"
	"testing/quick"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Job ID Codec", func() {
	DescribeTable("encoding job IDs",
		func(input, output string) {
			enc := Encode(input)
			Expect(enc).To(Equal(output))
			Expect(validRuncID(enc)).To(BeTrue())
		},
		Entry("empty string", "", "bpm-"),
		Entry("underscore", "_", "bpm-_"),
		Entry("classic name", "test-server", "bpm-test-server"),
		Entry("classic name with process", "test-server.alt-test-server", "bpm-test-server.2ealt-test-server"),
		Entry("goofy name", "test-server!@*&", "bpm-test-server.21.40.2a.26"),
		Entry("underscore name", "test_server", "bpm-test_server"),
	)

	DescribeTable("decoding job IDs",
		func(input, output string) {
			dec, err := Decode(input)
			Expect(err).NotTo(HaveOccurred())
			Expect(dec).To(Equal(output))
		},
		Entry("empty string", "bpm-", ""),
		Entry("classic name", "bpm-test-server", "test-server"),
		Entry("classic name with process", "bpm-test-server.2ealt-test-server", "test-server.alt-test-server"),
		Entry("goofy name", "bpm-test-server.21.40.2a.26", "test-server!@*&"),
		Entry("underscore name", "bpm-test_server", "test_server"),
	)

	DescribeTable("invalid job IDs",
		func(input string) {
			_, err := Decode(input)
			Expect(err).To(HaveOccurred())
		},
		Entry("no prefix", "unknown"),
		Entry("invalid escape codes", "bpm-."),
	)

	DescribeTable("edge cases",
		func(input string) {
			enc := Encode(input)
			Expect(validRuncID(enc)).To(BeTrue())

			dec, err := Decode(enc)
			Expect(err).NotTo(HaveOccurred())
			Expect(dec).To(Equal(input))
		},
		Entry("tiny literals", "someinput\x06aroundtheproblem"),
	)

	It("can successfully roundtrip job IDs no matter what string contents they contain", func() {
		rt := func(name string) bool {
			enc := Encode(name)
			Expect(validRuncID(enc)).To(BeTrue())

			dec, err := Decode(enc)
			Expect(err).NotTo(HaveOccurred())
			return dec == name
		}
		Expect(quick.Check(rt, nil)).To(Succeed())
	})
})

var idRegex = regexp.MustCompile(`^[\w+-.]+$`)

// https://github.com/opencontainers/runc/blob/e4fa8a457544ca646e02e60d124aebb0bb7f52ad/libcontainer/factory_linux.go#L373.
func validRuncID(id string) bool {
	return idRegex.MatchString(id) && string(os.PathSeparator)+id == lexicallyCleanPath(string(os.PathSeparator)+id)
}

// https://github.com/opencontainers/runc/blob/42a1e19d6788fde798fa960f047afbffbc319f8e/internal/pathrs/path.go#L45-L66
func lexicallyCleanPath(path string) string {
	// If the path isn't absolute, we need to do more processing to fix paths
	// such as "../../../../<etc>/some/path". We also shouldn't convert absolute
	// paths to relative ones.
	path = filepath.Clean(string(os.PathSeparator) + path)

	path, err := filepath.Rel(string(os.PathSeparator), path)
	Expect(err).NotTo(HaveOccurred()) // This can't fail, as (by definition) all paths are relative to root.

	return path
}
