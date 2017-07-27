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

package bpmlicensecheck

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

const copyrightHeader = "Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved."

var exceptions = []string{
	".gitignore",
	"fake_",
	".yml",
	"tags",
}

func Check(directory string) ([]string, error) {
	var incompleteFiles []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		for _, exception := range exceptions {
			if strings.Contains(path, exception) {
				return nil
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		r := bufio.NewReader(f)
		line, _, err := r.ReadLine()
		if err != nil {
			return err
		}

		if !strings.Contains(string(line), copyrightHeader) {
			incompleteFiles = append(incompleteFiles, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return incompleteFiles, nil
}
