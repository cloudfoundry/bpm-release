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
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"bpm/mount"
)

const (
	cgroupRoot       = "/sys/fs/cgroup"
	legacyCgroupRoot = "/cgroup/bpm"
)

// Needed for legacy cgroups migration. Not used in new code.
var subsystems = []string{"blkio", "cpu", "cpuacct", "cpuset", "devices", "freezer", "hugetlb", "memory", "perf_event", "pids"}

func Setup() error {
	mnts, err := mount.Mounts()
	if err != nil {
		return err
	}
	if err := mountCgroupTmpfsIfNotPresent(mnts); err != nil {
		return err
	}

	subsystems, err := EnabledSubsystems()
	if err != nil {
		return err
	}

	for _, sub := range subsystems {
		group, err := SubsystemGrouping(sub)
		if err != nil {
			return err
		}
		if err := mountCgroupSubsystem(group); err != nil {
			return err
		}
	}

	// TODO(jm): This exists for the legacy code cgroup mount point
	// This should be deleted when v1 is released.
	err = chmodLegacyMountPointIfPresent()
	if err != nil {
		return err
	}

	return nil
}

// SubsystemGrouping fetches the parent cgroup grouping (if any) for a
// particular subsystem.
func SubsystemGrouping(subsystem string) (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if os.IsNotExist(err) {
		// If the current process is not in a cgroup then we can do as we
		// please. We do not mount cgroups together.
		return subsystem, nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()

	// If the current process is in a cgroup then we need to match the
	// grouping of the parent cgroup.
	return subsystemGrouping(f, subsystem)
}

func subsystemGrouping(f io.Reader, subsystem string) (string, error) {
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		fields := strings.Split(line, ":")
		grouping := fields[1]
		subs := strings.Split(grouping, ",")
		if containsElement(subs, subsystem) {
			return grouping, nil
		}
	}
	if err := s.Err(); err != nil {
		return "", err
	}

	// If the current process isn't in a cgroup of the subsystem's type
	// then we don't need to match anything.
	return subsystem, nil
}

// EnabledSubsystems returns the cgroup subsystems which are enabled in the
// running kernel.
func EnabledSubsystems() ([]string, error) {
	f, err := os.Open("/proc/cgroups")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return enabledSubsystems(f)
}

func enabledSubsystems(f io.Reader) ([]string, error) {
	var subs []string

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()

		// Skip header
		if strings.HasPrefix(line, "#") {
			continue
		}

		var (
			name    string
			ignored int
			enabled bool
		)
		if _, err := fmt.Sscanf(s.Text(), "%s %d %d %t", &name, &ignored, &ignored, &enabled); err != nil {
			return nil, err
		}
		if enabled {
			subs = append(subs, name)
		}
	}

	if err := s.Err(); err != nil {
		return nil, err
	}

	return subs, nil
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

func mountCgroupSubsystem(subsystem string) error {
	mountPoint := filepath.Join(cgroupRoot, subsystem)
	if err := os.MkdirAll(mountPoint, 0755); err != nil {
		return err
	}
	if err := os.Chmod(mountPoint, 0755); err != nil {
		return err
	}

	err := mount.Mount("cgroup", mountPoint, "cgroup", 0, subsystem)
	switch err {
	// EBUSY is returned if the mountpoint already has something mounted on it
	case unix.EBUSY, nil:
		return nil
	default:
		return err
	}
}

func containsElement(elements []string, element string) bool {
	for _, e := range elements {
		if e == element {
			return true
		}
	}

	return false
}

func chmodLegacyMountPointIfPresent() error {
	err := chmodIfExists(legacyCgroupRoot)
	if err != nil {
		return err
	}

	for _, subsystem := range subsystems {
		err := chmodIfExists(filepath.Join(legacyCgroupRoot, subsystem))
		if err != nil {
			return err
		}
	}

	return nil
}

func chmodIfExists(path string) error {
	err := os.Chmod(path, 0755)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
