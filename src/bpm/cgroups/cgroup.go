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

package cgroups

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"

	"bpm/mount"
)

const cgroupFilesystem = "cgroup"

var (
	cgroupRoot = filepath.Join("/sys", "fs", "cgroup")
	subsystems = []string{"blkio", "cpu", "cpuacct", "cpuset", "devices", "freezer", "hugetlb", "memory", "perf_event", "pids"}
)

func Setup() error {
	mnts, err := mount.Mounts()
	if err != nil {
		return err
	}

	err = mountCgroupTmpfsIfNotPresent(mnts)
	if err != nil {
		return err
	}

	for _, subsystem := range subsystems {
		err := mountCgroupSubsystemIfNotPresent(mnts, subsystem)
		if err != nil {
			return err
		}
	}

	return nil
}

func mountCgroupTmpfsIfNotPresent(mnts []mount.Mnt) error {
	for _, mnt := range mnts {
		if mnt.MountPoint == cgroupRoot {
			return nil
		}
	}

	err := os.MkdirAll(cgroupRoot, 0755)
	if err != nil {
		return err
	}

	return mount.Mount("tmpfs", cgroupRoot, "tmpfs", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV, "mode=755")
}

func mountCgroupSubsystemIfNotPresent(mnts []mount.Mnt, subsystem string) error {
	for _, mnt := range mnts {
		if mnt.Filesystem == cgroupFilesystem && containsElement(mnt.Options, subsystem) {
			return nil
		}
	}

	mountPoint := filepath.Join(cgroupRoot, subsystem)
	err := os.MkdirAll(mountPoint, 0755)
	if err != nil {
		return err
	}

	err = os.Chmod(mountPoint, 0755)
	if err != nil {
		return err
	}

	return mount.Mount(cgroupFilesystem, mountPoint, cgroupFilesystem, 0, subsystem)
}

func containsElement(elements []string, element string) bool {
	for _, e := range elements {
		if e == element {
			return true
		}
	}

	return false
}
