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

package main_test

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"testing"
)

func TestBpm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

const bpmTmpDir = "/bpmtmp"

var (
	bpmPath string
)

var _ = SynchronizedBeforeSuite(func() []byte {
	bpmPath, err := gexec.Build("bpm")
	Expect(err).NotTo(HaveOccurred())

	err = os.MkdirAll(bpmTmpDir, 0755)
	Expect(err).NotTo(HaveOccurred())

	return []byte(bpmPath)
}, func(data []byte) {
	bpmPath = string(data)
	SetDefaultEventuallyTimeout(2 * time.Second)
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	err := os.RemoveAll(bpmTmpDir)
	Expect(err).NotTo(HaveOccurred())

	gexec.CleanupBuildArtifacts()
})
