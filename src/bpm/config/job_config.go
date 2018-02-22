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

package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type JobConfig struct {
	Processes []*ProcessConfig `yaml:"processes"`
}

type ProcessConfig struct {
	Name              string            `yaml:"name"`
	Executable        string            `yaml:"executable"`
	Args              []string          `yaml:"args"`
	Env               map[string]string `yaml:"env"`
	AdditionalVolumes []Volume          `yaml:"additional_volumes"`
	Capabilities      []string          `yaml:"capabilities"`
	EphemeralDisk     bool              `yaml:"ephemeral_disk"`
	Hooks             *Hooks            `yaml:"hooks,omitempty"`
	Limits            *Limits           `yaml:"limits"`
	PersistentDisk    bool              `yaml:"persistent_disk"`
	WorkDir           string            `yaml:"workdir"`
}

type Limits struct {
	Memory    *string `yaml:"memory"`
	OpenFiles *uint64 `yaml:"open_files"`
	Processes *int64  `yaml:"processes"`
}

type Hooks struct {
	PreStart string `yaml:"pre_start"`
}

type Volume struct {
	Path            string `yaml:"path"`
	Writable        bool   `yaml:"writable"`
	AllowExecutions bool   `yaml:"allow_executions"`
}

func ParseJobConfig(configPath string) (*JobConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := JobConfig{}

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

var validCaps = map[string]bool{
	"NET_BIND_SERVICE": true,
}

func (c *JobConfig) Validate() error {
	for _, v := range c.Processes {
		if err := v.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *ProcessConfig) Validate() error {
	if c.Name == "" {
		return errors.New("invalid config: name")
	}

	if c.Executable == "" {
		return errors.New("invalid config: executable")
	}

	for _, vol := range c.AdditionalVolumes {
		volCleaned := filepath.Clean(vol.Path)
		if volCleaned != vol.Path {
			return fmt.Errorf("volume path must be canonical, expected %s but got %s", volCleaned, vol.Path)
		}

		if !pathIsIn(validDataVolumePrefix, volCleaned) && !pathIsIn(validStoreVolumePrefix, volCleaned) {
			return fmt.Errorf(
				"invalid volume path: %s must be within (%s,%s)",
				vol.Path,
				validDataVolumePrefix,
				validStoreVolumePrefix,
			)
		}
	}

	for _, capabilities := range c.Capabilities {
		if _, ok := validCaps[capabilities]; !ok {
			return fmt.Errorf(
				"invalid capability: %s",
				capabilities,
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
