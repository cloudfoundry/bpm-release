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
)

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

	// runc spec only allows `^[\w+-\.]+$`
	// https://github.com/opencontainers/runc/blob/master/libcontainer/factory_linux.go
	return Encode(containerID)
}

func Encode(containerID string) string {
	enc := base32.StdEncoding
	enc = enc.WithPadding('-')
	return enc.EncodeToString([]byte(containerID))
}

func Decode(containerID string) (string, error) {
	enc := base32.StdEncoding
	enc = enc.WithPadding('-')
	data, err := enc.DecodeString(containerID)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
