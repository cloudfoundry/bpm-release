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
	"slices"
	"strings"

	"github.com/moby/sys/mountinfo"
	"github.com/opencontainers/cgroups"
	"golang.org/x/sys/unix"
)

const cgroupRoot = "/sys/fs/cgroup"

func Setup() error {
	if cgroups.IsCgroup2UnifiedMode() {
		return nil
	}

	mounts, err := mountinfo.GetMounts(mountinfo.ParentsFilter(cgroupRoot))
	if err != nil {
		return fmt.Errorf("unable to retrieve mounts: %s", err)
	}
	if err = mountCgroupTmpfsIfNotPresent(mounts); err != nil {
		return fmt.Errorf("unable to mount cgroup tmpfs: %s", err)
	}

	subsystems, err := cgroups.GetAllSubsystems()
	if err != nil {
		return fmt.Errorf("unable to retrieve cgroup subsystems: %s", err)
	}

	for _, sub := range subsystems {
		var group string
		group, err = subsystemGrouping(sub)
		if err != nil {
			return fmt.Errorf("unable to retrieve subsystem grouping for %s: %s", sub, err)
		}

		err = mountCgroupSubsystem(group)
		if err != nil {
			return fmt.Errorf("unable to mount subsystem for %s: %s", sub, err)
		}
	}

	return nil
}

func subsystemGrouping(subsystem string) (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if os.IsNotExist(err) {
		// If the current process is not in a cgroup then we can do as we
		// please. We do not mount cgroups together.
		return subsystem, nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	// If the current process is in a cgroup then we need to match the
	// grouping of the parent cgroup.
	return subsystemGroupingFromProcCgroup(f, subsystem)
}

func subsystemGroupingFromProcCgroup(f io.Reader, subsystem string) (string, error) {
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		fields := strings.Split(line, ":")
		grouping := fields[1]
		subs := strings.Split(grouping, ",")
		if slices.Contains(subs, subsystem) {
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

// SelfCgroupPath returns the cgroup v2 unified-mode path of the calling
// process by reading /proc/self/cgroup. Returns an error if no unified-mode
// entry (0::<path>) is found.
func SelfCgroupPath() (string, error) {
	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", fmt.Errorf("opening /proc/self/cgroup: %w", err)
	}
	defer f.Close() //nolint:errcheck
	return selfCgroupPathFromReader(f)
}

func selfCgroupPathFromReader(r io.Reader) (string, error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		line := s.Text()
		// cgroup v2 unified-mode line: "0::<path>"
		if strings.HasPrefix(line, "0::") {
			return strings.TrimRight(strings.TrimPrefix(line, "0::"), "\r\n"), nil
		}
	}
	if err := s.Err(); err != nil {
		return "", fmt.Errorf("reading /proc/self/cgroup: %w", err)
	}
	return "", fmt.Errorf("no cgroup v2 entry found in /proc/self/cgroup")
}

// ToSystemdCgroupsPath converts an absolute cgroup v2 unified-mode path and
// a container ID into the "slice:prefix:name" format expected by runc's
// systemd cgroup driver. It extracts the parent slice and a unique identifier
// from the first non-slice component (e.g., the garden container scope) to
// ensure the resulting scope name is unique per warden container.
//
// Example:
//
//	selfPath = "/system.slice/garden-abc.scope/monit.service"
//	containerID = "bpm-uaa"
//	result = "system.slice:garden-abc-scope-bpm:bpm-uaa"
func ToSystemdCgroupsPath(selfPath, containerID string) string {
	parts := strings.Split(strings.TrimLeft(selfPath, "/"), "/")

	slice := "system.slice" // fallback if no .slice found
	uniquePart := ""

	for i, part := range parts {
		if strings.HasSuffix(part, ".slice") {
			slice = part
			if i+1 < len(parts) {
				normalized := normalizeForSystemdName(parts[i+1])
				if normalized != "" {
					uniquePart = normalized + "-"
				}
			}
			break
		}
	}

	// If no .slice was found, use the first non-empty path element as a
	// uniqueness anchor so the scope name still reflects the host context.
	if uniquePart == "" {
		for _, part := range parts {
			if normalized := normalizeForSystemdName(part); normalized != "" {
				uniquePart = normalized + "-"
				break
			}
		}
	}

	return fmt.Sprintf("%s:%sbpm:%s", slice, uniquePart, containerID)
}

// normalizeForSystemdName replaces characters invalid in systemd unit name
// components with dashes. Valid characters are alphanumeric, '-', and '_'.
func normalizeForSystemdName(s string) string {
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' {
			b.WriteRune(c)
		} else {
			b.WriteRune('-')
		}
	}
	return b.String()
}

func mountCgroupTmpfsIfNotPresent(mountInfos []*mountinfo.Info) error {
	for _, mnt := range mountInfos {
		if mnt.Mountpoint == cgroupRoot {
			return nil
		}
	}

	err := os.MkdirAll(cgroupRoot, 0755)
	if err != nil {
		return err
	}

	return unix.Mount("tmpfs", cgroupRoot, "tmpfs", unix.MS_NOSUID|unix.MS_NOEXEC|unix.MS_NODEV, "mode=755")
}

func mountCgroupSubsystem(subsystem string) error {
	mountpoint := filepath.Join(cgroupRoot, subsystem)
	if _, err := os.Stat(mountpoint); os.IsNotExist(err) {
		if err := os.MkdirAll(mountpoint, 0755); err != nil {
			return err
		}
		if err := os.Chmod(mountpoint, 0755); err != nil {
			return err
		}
	}

	err := unix.Mount("cgroup", mountpoint, "cgroup", 0, subsystem)
	switch err {
	// EBUSY is returned if the mountpoint already has something mounted on it
	case unix.EBUSY, nil:
		return nil
	default:
		return fmt.Errorf("unable to mount %s: %w", mountpoint, err)
	}
}
