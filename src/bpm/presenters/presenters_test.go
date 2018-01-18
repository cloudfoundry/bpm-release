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

package presenters_test

import (
	"bpm/config"
	"bpm/models"
	"bpm/presenters"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("Presenters", func() {
	Describe("PresentJobs", func() {
		var (
			processes []*models.Process
			output    *gbytes.Buffer
		)

		BeforeEach(func() {
			processes = []*models.Process{
				{Name: config.Encode("job-process-2"), Pid: 23456, Status: "created"},
				{Name: config.Encode("job-process-1"), Pid: 34567, Status: "running"},
				{Name: config.Encode("job-process-3"), Pid: 0, Status: "failed"},
			}

			output = gbytes.NewBuffer()
		})

		It("prints the jobs in a table", func() {
			Expect(presenters.PrintJobs(processes, output)).To(Succeed())
			Expect(output).Should(gbytes.Say("Name\\s+Pid\\s+Status"))
			Expect(output).Should(gbytes.Say(fmt.Sprintf("%s\\s+%d\\s+%s", "job-process-2", 23456, "created")))
			Expect(output).Should(gbytes.Say(fmt.Sprintf("%s\\s+%d\\s+%s", "job-process-1", 34567, "running")))
			Expect(output).Should(gbytes.Say(fmt.Sprintf("%s\\s+%s\\s+%s", "job-process-3", "-", "failed")))
		})
	})
})
