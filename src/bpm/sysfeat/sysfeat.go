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

// Package sysfeat fetches information about the host system which can be used
// to enable if disable specific features depending on what's supported.
package sysfeat

import (
	"os"
	"path/filepath"

	"github.com/opencontainers/runc/libcontainer/cgroups"
)

const (
	swapPath = "memory.memsw.limit_in_bytes"
)

// Features contains information about what features the host system supports.
type Features struct {
	// Whether the system supports limiting the swap space of a process or not.
	SwapLimitSupported bool
}

func Fetch() (*Features, error) {
	mountpoint, err := cgroups.FindCgroupMountpoint("memory")
	if err != nil {
		return nil, err
	}

	return &Features{
		SwapLimitSupported: swapLimitSupported(mountpoint),
	}, nil
}

func swapLimitSupported(mount string) bool {
	_, err := os.Stat(filepath.Join(mount, swapPath))
	return err == nil
}
