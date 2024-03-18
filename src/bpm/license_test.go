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

package bpm_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLicenses(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "License Suite")
}

var checks = []string{
	"Copyright",
	"CloudFoundry.org Foundation, Inc.",
	"www.apache.org",
}

var _ = Describe("our go source code", func() {
	It("has license headers", func() {
		var missingHeader []string

		err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Do not check our vendored code
			if strings.HasPrefix(path, "vendor/") {
				return filepath.SkipDir
			}

			// Ignore directories
			if info.IsDir() {
				return nil
			}

			// Only check our Go code
			if filepath.Ext(path) != ".go" {
				return nil
			}

			// Skip generated code
			if strings.HasPrefix(filepath.Base(path), "fake_") {
				return nil
			}

			bs, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// The check strings above should be near the top of the file.
			bs = bs[:512]

			for _, check := range checks {
				if !strings.Contains(string(bs), check) {
					missingHeader = append(missingHeader, path)
					break
				}
			}

			return nil
		})

		Expect(err).NotTo(HaveOccurred())
		Expect(missingHeader).To(BeEmpty(), "These files are missing license headers!")
	})
})
