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

	yaml "gopkg.in/yaml.v2"

	"bpm/bosh"
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
	Unsafe            *Unsafe           `yaml:"unsafe"`
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
	MountOnly       bool   `yaml:"mount_only"`
	Shared          bool   `yaml:"shared"`
}

type Unsafe struct {
	Privileged          bool     `yaml:"privileged"`
	UnrestrictedVolumes []Volume `yaml:"unrestricted_volumes"`
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

	return &cfg, nil
}

func (c *JobConfig) Validate(boshEnv *bosh.Env, defaultVolumes []string) error {
	for _, v := range c.Processes {
		if err := v.Validate(boshEnv, defaultVolumes); err != nil {
			return err
		}
	}

	return nil
}

func (c *ProcessConfig) Validate(boshEnv *bosh.Env, defaultVolumes []string) error {
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

		if contains(defaultVolumes, volCleaned) {
			return fmt.Errorf(
				"invalid volume path: %s cannot conflict with default job data or store directories",
				vol.Path,
			)
		}

		if !pathIsIn(volCleaned, boshEnv.Root().External()) {
			return fmt.Errorf(
				"invalid volume path: %s must be within %s",
				vol.Path,
				boshEnv.Root().External(),
			)
		}
	}

	return nil
}

func (c *ProcessConfig) AddVolumes(
	volumes []string,
	boshEnv *bosh.Env,
	defaultVolumes []string,
) error {
	for _, volume := range volumes {
		fields := strings.Split(volume, ":")

		if len(fields) > 2 {
			return fmt.Errorf("invalid volume definition (format: <path>[:<options>]): %s", volume)
		}

		v := Volume{
			Path: fields[0],
		}

		if len(fields) == 2 {
			options := strings.Split(fields[1], ",")
			for _, option := range options {
				switch option {
				case "writable":
					v.Writable = true
				case "mount_only":
					v.MountOnly = true
				case "allow_executions":
					v.AllowExecutions = true
				case "shared":
					v.Shared = true
				default:
					return fmt.Errorf("invalid volume option: %s", option)
				}
			}
		}

		c.AdditionalVolumes = append(c.AdditionalVolumes, v)
	}

	return c.Validate(boshEnv, defaultVolumes)
}

// AddEnvVars allows additional environment variables to be added to a process
// configuration after parsing the configuration file. The environment
// variables take the form of "KEY=VALUE". If a key is specified multiple times
// then the last valeu wins.
func (c *ProcessConfig) AddEnvVars(
	env []string,
	boshEnv *bosh.Env,
	defaultVolumes []string,
) error {
	if c.Env == nil {
		c.Env = map[string]string{}
	}
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) < 2 {
			return fmt.Errorf("invalid envionment variable definition (format should be KEY=value): %q", e)
		}
		key, value := parts[0], parts[1]
		c.Env[key] = value
	}
	return c.Validate(boshEnv, defaultVolumes)
}

func contains(elements []string, s string) bool {
	for _, elem := range elements {
		if s == elem {
			return true
		}
	}

	return false
}

func pathIsIn(path string, prefixes ...string) bool {
	volParts := strings.Split(path, "/")

	for _, prefix := range prefixes {
		validParts := strings.Split(prefix, "/")

		if len(volParts) <= len(validParts) {
			continue
		}

		match := true
		for i, validPart := range validParts {
			if volParts[i] != validPart {
				match = false
				break
			}
		}

		if !match {
			continue
		}

		return true
	}

	return false
}
