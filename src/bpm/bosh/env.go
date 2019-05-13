// Copyright (C) 2018-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package bosh

import (
	"io/ioutil"
	"path/filepath"
)

// Env represents a BOSH BPM environment. This is not a concept outside BPM but
// allows us to isolate BPM invocations from one another so that integration
// tests can be made concurrent.
type Env struct {
	root string
}

// NewEnv creates a new environment with a particular directory as its root. If
// root is empty then the DefaultRoot will be used.
func NewEnv(root string) *Env {
	if root == "" {
		root = DefaultRoot
	}

	return &Env{
		root: root,
	}
}

// JobNames returns a list of all the job names inside a BOSH environment.
func (e *Env) JobNames() []string {
	var jobs []string

	fileInfos, err := ioutil.ReadDir(filepath.Join(e.root, "jobs"))
	if err != nil {
		return jobs
	}

	for _, info := range fileInfos {
		jobs = append(jobs, info.Name())
	}

	return jobs
}

// Root returns a Path representation of the environment's root. It is
// typically not useful by itself but can be appended to with Path.Join().
func (e *Env) Root() Path {
	return Path{root: e.root}
}

// DataDir returns a Path representation of the directory where a job should
// store its ephemeral data.
func (e *Env) DataDir(job string) Path {
	return Path{root: e.root, dir: filepath.Join("data", job)}
}

// StoreDir returns a Path representation of the directory where a job should
// store its persistent data.
func (e *Env) StoreDir(job string) Path {
	return Path{root: e.root, dir: filepath.Join("store", job)}
}

// JobDir returns a Path representation of the directory where a job should
// can find its templated BOSH configuration.
func (e *Env) JobDir(job string) Path {
	return Path{root: e.root, dir: filepath.Join("jobs", job)}
}

// RunDir returns a Path representation of the directory where a job should
// store any sockets or PID files.
func (e *Env) RunDir(job string) Path {
	return Path{root: e.root, dir: filepath.Join("sys", "run", job)}
}

// LogDir returns a Path representation of the directory where a job should
// write any additional log files.
func (e *Env) LogDir(job string) Path {
	return Path{root: e.root, dir: filepath.Join("sys", "log", job)}
}

// PackageDir returns a Path representation of the global directory where all BOSH
// packages are stored.
func (e *Env) PackageDir() Path {
	return Path{root: e.root, dir: "packages"}
}

// DataPackageDir returns a Path representation of the global directory where
// all BOSH packages are actually stored (PackageDir is typically a symlink to
// this).
func (e *Env) DataPackageDir() Path {
	return Path{root: e.root, dir: filepath.Join("data", "packages")}
}
