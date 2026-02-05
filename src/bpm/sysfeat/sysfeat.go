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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/opencontainers/cgroups"
)

const (
	swapPathCgroup1 = "memory.memsw.limit_in_bytes"
	swapPathCgroup2 = "memory.swap.max"

	unifiedMountpoint = "/sys/fs/cgroup"
	hybridMountpoint  = "/sys/fs/cgroup/unified"
)

// goArchToKernelArch maps Go's GOARCH values to Linux kernel architecture
// names as returned by `uname -m`. This mapping is used to detect when
// a binary is running under architecture emulation (e.g., x86_64 binaries
// on ARM64 kernels via Rosetta).
var goArchToKernelArch = map[string]string{
	"amd64": "x86_64",
	"386":   "i686",
	"arm64": "aarch64",
	"arm":   "armv7l",
}

// Features contains information about what features the host system supports.
type Features struct {
	// Whether the system supports limiting the swap space of a process or not.
	SwapLimitSupported bool
	// Whether the system supports seccomp BPF filtering. This may be false in
	// environments with architecture emulation (e.g., x86_64 binaries running
	// on ARM64 kernels via Rosetta).
	SeccompSupported bool
}

func Fetch() (*Features, error) {
	supported, err := swapLimitSupported()
	if err != nil {
		return nil, err
	}

	return &Features{
		SwapLimitSupported: supported,
		SeccompSupported:   seccompSupported(),
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

// seccompSupported checks whether seccomp BPF filtering is supported in the
// current environment. It returns false when running in a container with
// architecture emulation (e.g., x86_64 binaries on ARM64 kernels).
func seccompSupported() bool {
	// Allow override to force seccomp enabled even in emulated environments
	if os.Getenv("BPM_DISABLE_SECCOMP_DETECTION") != "" {
		return true
	}

	// If not in a container, seccomp works normally
	if !isRunningInContainer() {
		return true
	}

	// Check if Go binary architecture matches kernel architecture
	goArch := runtime.GOARCH      // e.g., "amd64"
	kernelArch := getKernelArch() // e.g., "x86_64"

	expectedKernelArch, ok := goArchToKernelArch[goArch]
	if !ok {
		// Unknown architecture mapping, assume seccomp works (conservative)
		return true
	}

	// If architectures don't match, we're under emulation
	// Seccomp BPF filters won't work
	if kernelArch != expectedKernelArch {
		return false
	}

	return true
}

// isRunningInContainer checks whether the current process is running inside
// a container environment.
func isRunningInContainer() bool {
	// Check for /.dockerenv
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check systemd-detect-virt -c
	cmd := exec.Command("systemd-detect-virt", "-c")
	output, err := cmd.Output()
	if err == nil {
		result := strings.TrimSpace(string(output))
		if result != "none" && result != "" {
			return true
		}
	}

	// Check /proc/1/cgroup for container indicators
	data, err := os.ReadFile("/proc/1/cgroup")
	if err == nil {
		content := string(data)
		if strings.Contains(content, "docker") ||
			strings.Contains(content, "lxc") ||
			strings.Contains(content, "kubepods") {
			return true
		}
	}

	return false
}

// getKernelArch returns the kernel architecture using uname -m.
func getKernelArch() string {
	cmd := exec.Command("uname", "-m")
	output, err := cmd.Output()
	if err != nil {
		// If we can't determine the kernel arch, assume it matches (conservative)
		return ""
	}
	return strings.TrimSpace(string(output))
}
