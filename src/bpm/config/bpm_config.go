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
	"encoding/base32"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

const ContainerPrefix = "bpm-"

func RuncPath(boshRoot string) string {
	return filepath.Join(boshRoot, "packages", "bpm", "bin", "runc")
}

func BundlesRoot(boshRoot string) string {
	return filepath.Join(boshRoot, "data", "bpm", "bundles")
}

func RuncRoot(boshRoot string) string {
	return filepath.Join(boshRoot, "data", "bpm", "runc")
}

type BPMConfig struct {
	boshRoot string
	jobName  string
	procName string
}

func NewBPMConfig(boshRoot, jobName, procName string) *BPMConfig {
	return &BPMConfig{
		boshRoot: boshRoot,
		jobName:  jobName,
		procName: procName,
	}
}

func (c *BPMConfig) JobName() string {
	return c.jobName
}

func (c *BPMConfig) ProcName() string {
	return c.procName
}

func (c *BPMConfig) DataDir() string {
	return filepath.Join(c.boshRoot, "data", c.jobName)
}

func (c *BPMConfig) StoreDir() string {
	return filepath.Join(c.boshRoot, "store", c.jobName)
}

func (c *BPMConfig) SocketDir() string {
	return filepath.Join(c.boshRoot, "sys", "run", c.jobName)
}

func (c *BPMConfig) TempDir() string {
	return filepath.Join(c.DataDir(), "tmp")
}

func (c *BPMConfig) LogDir() string {
	return filepath.Join(c.boshRoot, "sys", "log", c.jobName)
}

func (c *BPMConfig) Stdout() string {
	return filepath.Join(c.LogDir(), fmt.Sprintf("%s.stdout.log", c.procName))
}

func (c *BPMConfig) Stderr() string {
	return filepath.Join(c.LogDir(), fmt.Sprintf("%s.stderr.log", c.procName))
}

func (c *BPMConfig) PidDir() string {
	return filepath.Join(c.boshRoot, "sys", "run", "bpm", c.jobName)
}

func (c *BPMConfig) PidFile() string {
	return filepath.Join(c.PidDir(), fmt.Sprintf("%s.pid", c.procName))
}

func (c *BPMConfig) LockFile() string {
	return filepath.Join(c.PidDir(), fmt.Sprintf("%s.lock", c.procName))
}

func (c *BPMConfig) PackageDir() string {
	return filepath.Join(c.boshRoot, "packages")
}

func (c *BPMConfig) DataPackageDir() string {
	return filepath.Join(c.boshRoot, "data", "packages")
}

func (c *BPMConfig) JobDir() string {
	return filepath.Join(c.boshRoot, "jobs", c.jobName)
}

func (c *BPMConfig) JobConfig() string {
	return filepath.Join(c.JobDir(), "config", "bpm.yml")
}

func (c *BPMConfig) DefaultVolumes() []string {
	return []string{c.DataDir(), c.StoreDir()}
}

func (c *BPMConfig) ParseJobConfig() (*JobConfig, error) {
	cfg, err := ParseJobConfig(c.JobConfig())
	if err != nil {
		return nil, err
	}

	err = cfg.Validate(c.boshRoot, c.DefaultVolumes())
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *BPMConfig) BPMLog() string {
	return filepath.Join(c.LogDir(), "bpm.log")
}

func (c *BPMConfig) BundlePath() string {
	return filepath.Join(BundlesRoot(c.boshRoot), c.jobName, c.procName)
}

func (c *BPMConfig) RootFSPath() string {
	return filepath.Join(c.BundlePath(), "rootfs")
}

func (c *BPMConfig) ContainerID() string {
	var containerID string

	if c.jobName == c.procName {
		containerID = c.jobName
	} else {
		containerID = fmt.Sprintf("%s.%s", c.jobName, c.procName)
	}

	return ContainerPrefix + Encode(containerID)
}

// Encode encodes a containerID returning a valid runc ID matching the pattern `^[\w+-\.]+$`
// see https://github.com/opencontainers/runc/blob/079817cc26ec5292ac375bb9f47f373d33574949/libcontainer/factory_linux.go#L32
// Every substring not complying with the runc format gets base32 encoded delimited by `+`
func Encode(containerID string) string {
	invalidRuncSubstring := regexp.MustCompile(`[^\w,-\.]+`)
	return invalidRuncSubstring.ReplaceAllStringFunc(containerID, base32EncodeWithDelimiter)
}

func Decode(containerID string) (string, error) {
	if containerID == "" {
		return "", nil
	}

	base32Encoded := regexp.MustCompile(`\+[^+]+\+`)
	decoded := base32Encoded.ReplaceAllStringFunc(containerID, base32DecodeWithDelimiter)
	if decoded == "" {
		return "", fmt.Errorf("could not decode container ID (%s)", containerID)
	}
	return decoded, nil
}

func base32EncodeWithDelimiter(invalidSubstring string) string {
	enc := base32.StdEncoding
	enc = enc.WithPadding('-')
	encoded := enc.EncodeToString([]byte(invalidSubstring))
	return fmt.Sprintf("+%s+", encoded)
}

func base32DecodeWithDelimiter(encoded string) string {
	withoutDelimiter := strings.Trim(encoded, "+")
	enc := base32.StdEncoding
	enc = enc.WithPadding('-')
	decoded, err := enc.DecodeString(withoutDelimiter)
	if err != nil {
		return ""
	}
	return string(decoded)
}
