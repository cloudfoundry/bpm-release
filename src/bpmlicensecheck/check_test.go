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

package bpmlicensecheck_test

import (
	"bpmlicensecheck"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check", func() {
	It("check for licenses successfully", func() {
		gopath := os.Getenv("GOPATH")

		files, err := bpmlicensecheck.Check(".")
		Expect(err).NotTo(HaveOccurred())
		for _, pkg := range packages {
			newFiles, err := bpmlicensecheck.Check(filepath.Join(gopath, "src", pkg))
			Expect(err).NotTo(HaveOccurred())
			files = append(files, newFiles...)
		}

		Expect(files).To(HaveLen(0))
	})
})
