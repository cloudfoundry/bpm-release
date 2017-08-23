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

package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

type ProcessConfig struct {
	Executable     string   `yaml:"executable"`
	Args           []string `yaml:"args"`
	Env            []string `yaml:"env"`
	Limits         *Limits  `yaml:"limits"`
	Volumes        []string `yaml:"volumes"`
	Hooks          *Hooks   `yaml:"hooks"`
	AdditionalJobs []string `yaml:"additional_jobs"`
}

type Limits struct {
	Memory    *string `yaml:"memory"`
	OpenFiles *uint64 `yaml:"open_files"`
	Processes *int64  `yaml:"processes"`
}

type Hooks struct {
	PreStart string `yaml:"pre_start"`
}

func ParseProcessConfig(configPath string) (*ProcessConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := ProcessConfig{}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	err = cfg.Validate()
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

const (
	validDataVolumePrefix  = "/var/vcap/data"
	validStoreVolumePrefix = "/var/vcap/store"
)

func (c *ProcessConfig) Validate() error {
	if c.Executable == "" {
		return errors.New("invalid config: executable")
	}

	for _, vol := range c.Volumes {
		volCleaned := filepath.Clean(vol)
		if volCleaned != vol {
			return fmt.Errorf("volume path must be canonical, expected %s but got %s", volCleaned, vol)
		}

		if !pathIsIn(validDataVolumePrefix, volCleaned) && !pathIsIn(validStoreVolumePrefix, volCleaned) {
			return fmt.Errorf(
				"invalid volume path: %s must be within (%s,%s)",
				vol,
				validDataVolumePrefix,
				validStoreVolumePrefix,
			)
		}
	}

	return nil
}

func pathIsIn(prefix string, path string) bool {
	volParts := strings.Split(path, "/")
	validParts := strings.Split(prefix, "/")

	if len(volParts) <= len(validParts) {
		return false
	}

	for i, validPart := range validParts {
		if volParts[i] != validPart {
			return false
		}
	}

	return true
}
