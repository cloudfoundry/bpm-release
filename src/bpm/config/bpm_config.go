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
	"fmt"
	"path/filepath"

	"bpm/bosh"
	"bpm/jobid"
)

func RuncPath(env *bosh.Env) string {
	return env.Root().Join("packages", "bpm", "bin", "runc").External()
}

func BundlesRoot(env *bosh.Env) string {
	return env.Root().Join("data", "bpm", "bundles").External()
}

func RuncRoot(env *bosh.Env) string {
	return env.Root().Join("data", "bpm", "runc").External()
}

type BPMConfig struct {
	jobName  string
	procName string

	boshEnv *bosh.Env
}

func NewBPMConfig(boshEnv *bosh.Env, jobName, procName string) *BPMConfig {
	return &BPMConfig{
		jobName:  jobName,
		procName: procName,
		boshEnv:  boshEnv,
	}
}

func (c *BPMConfig) JobName() string {
	return c.jobName
}

func (c *BPMConfig) ProcName() string {
	return c.procName
}

func (c *BPMConfig) DataDir() bosh.Path {
	return c.boshEnv.DataDir(c.JobName())
}

func (c *BPMConfig) StoreDir() bosh.Path {
	return c.boshEnv.StoreDir(c.JobName())
}

func (c *BPMConfig) SocketDir() bosh.Path {
	return c.boshEnv.RunDir(c.JobName())
}

func (c *BPMConfig) TempDir() bosh.Path {
	return c.DataDir().Join("tmp")
}

func (c *BPMConfig) LogDir() bosh.Path {
	return c.boshEnv.LogDir(c.JobName())
}

func (c *BPMConfig) Stdout() bosh.Path {
	return c.LogDir().Join(fmt.Sprintf("%s.stdout.log", c.procName))
}

func (c *BPMConfig) Stderr() bosh.Path {
	return c.LogDir().Join(fmt.Sprintf("%s.stderr.log", c.procName))
}

func (c *BPMConfig) PidDir() bosh.Path {
	return c.boshEnv.RunDir("bpm").Join(c.JobName())
}

func (c *BPMConfig) PidFile() bosh.Path {
	return c.PidDir().Join(fmt.Sprintf("%s.pid", c.procName))
}

func (c *BPMConfig) LockFile() bosh.Path {
	return c.PidDir().Join(fmt.Sprintf("%s.lock", c.procName))
}

func (c *BPMConfig) PackageDir() bosh.Path {
	return c.boshEnv.PackageDir()
}

func (c *BPMConfig) DataPackageDir() bosh.Path {
	return c.boshEnv.DataPackageDir()
}

func (c *BPMConfig) JobDir() bosh.Path {
	return c.boshEnv.JobDir(c.JobName())
}

func (c *BPMConfig) JobConfig() string {
	return c.JobDir().Join(filepath.Join("config", "bpm.yml")).External()
}

func (c *BPMConfig) DefaultVolumes() []string {
	return []string{c.DataDir().External(), c.StoreDir().External()}
}

func (c *BPMConfig) ParseJobConfig() (*JobConfig, error) {
	cfg, err := ParseJobConfig(c.JobConfig())
	if err != nil {
		return nil, err
	}

	err = cfg.Validate(c.boshEnv, c.DefaultVolumes())
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *BPMConfig) BPMLog() string {
	return c.LogDir().Join("bpm.log").External()
}

func (c *BPMConfig) BundlePath() string {
	return filepath.Join(BundlesRoot(c.boshEnv), c.jobName, c.procName)
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

	return jobid.Encode(containerID)
}
