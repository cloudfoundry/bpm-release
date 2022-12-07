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

package handlers

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"

	"bpm/runc/specbuilder"
)

func MaskedPaths(w http.ResponseWriter, r *http.Request) {
	readablePaths := ""
	emptyFileOutput := regexp.MustCompile("0\\+0 records in\n0\\+0 records out")

	for _, path := range specbuilder.DefaultSpec().Linux.MaskedPaths {
		stat, err := os.Stat(path)
		if err != nil {
			continue
		}

		if stat.IsDir() {
			contents, err := os.ReadDir(path)

			if err != nil {
				readablePaths += fmt.Sprintf("Error reading directory: %s\n", path)
				continue
			}

			if len(contents) == 0 {
				continue
			}

			readablePaths += fmt.Sprintf("Unexpected readable directory contents: %s\n", path)
		} else {
			cmd := exec.Command("/bin/dd", fmt.Sprintf("if=%s", path))
			output, err := cmd.CombinedOutput()
			defer cmd.Process.Kill() // as dd could run indefinitely and cause issues with other tests...

			if err != nil {
				readablePaths += fmt.Sprintf("Error reading path: %s\n", path)
				continue
			}

			if emptyFileOutput.Match(output) {
				continue
			}

			readablePaths += fmt.Sprintf("Unexpected readable path: %s\n", path)
		}
	}

	fmt.Fprint(w, string(readablePaths))
}
