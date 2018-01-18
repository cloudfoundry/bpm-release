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

package bpmlicensecheck

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

var checks = []string{
	"Copyright",
	"CloudFoundry.org Foundation, Inc.",
	"www.apache.org",
}

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

		bs, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		// The check strings above should be near the top of the file.
		bs = bs[:512]

		for _, check := range checks {
			if !strings.Contains(string(bs), check) {
				incompleteFiles = append(incompleteFiles, path)
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return incompleteFiles, nil
}
