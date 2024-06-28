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
	swapPathCgroup1 = "memory.memsw.limit_in_bytes"
	swapPathCgroup2 = "memory.swap.max"

	unifiedMountpoint = "/sys/fs/cgroup"
	hybridMountpoint  = "/sys/fs/cgroup/unified"
)

// Features contains information about what features the host system supports.
type Features struct {
	// Whether the system supports limiting the swap space of a process or not.
	SwapLimitSupported bool
}

func Fetch() (*Features, error) {
	supported, err := swapLimitSupported()
	if err != nil {
		return nil, err
	}

	return &Features{
		SwapLimitSupported: supported,
	}, nil
}

func swapLimitSupported() (bool, error) {
	if cgroups.IsCgroup2UnifiedMode() {
		return swapLimitSupportedCgroup2()
	}

	return swapLimitSupportedCgroup1()
}

func swapLimitSupportedCgroup2() (bool, error) {
	mountpoint := unifiedMountpoint
	if cgroups.IsCgroup2HybridMode() {
		mountpoint = hybridMountpoint
	}

	if cgroups.PathExists(filepath.Join(mountpoint, swapPathCgroup2)) {
		return true, nil
	}

	return false, nil
}

func swapLimitSupportedCgroup1() (bool, error) {
	mountPoint, err := cgroups.FindCgroupMountpoint("", "memory")
	if err != nil {
		return false, err
	}

	_, err = os.Stat(filepath.Join(mountPoint, swapPathCgroup1))
	return err == nil, nil
}
