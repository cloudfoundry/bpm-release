// Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
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

package adapter

import (
	"bpm/bpm"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"code.cloudfoundry.org/bytefmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type RuncAdapter struct{}

func NewRuncAdapter() *RuncAdapter {
	return &RuncAdapter{}
}

func (a *RuncAdapter) CreateJobPrerequisites(
	systemRoot string,
	jobName string,
	cfg *bpm.Config,
	user specs.User,
) (string, *os.File, *os.File, error) {
	bpmPidDir := filepath.Join(systemRoot, "sys", "run", "bpm", jobName)
	jobLogDir := filepath.Join(systemRoot, "sys", "log", jobName)
	stdoutFileLocation := filepath.Join(jobLogDir, fmt.Sprintf("%s.out.log", cfg.Name))
	stderrFileLocation := filepath.Join(jobLogDir, fmt.Sprintf("%s.err.log", cfg.Name))
	dataDir := filepath.Join(systemRoot, "data", jobName, cfg.Name)

	err := os.MkdirAll(bpmPidDir, 0700)
	if err != nil {
		return "", nil, nil, err
	}

	err = os.MkdirAll(jobLogDir, 0750)
	if err != nil {
		return "", nil, nil, err
	}
	err = os.Chown(jobLogDir, int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	stdout, err := createFileFor(stdoutFileLocation, int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	stderr, err := createFileFor(stderrFileLocation, int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	err = os.MkdirAll(dataDir, 0700)
	if err != nil {
		return "", nil, nil, err
	}
	err = os.Chown(dataDir, int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	return bpmPidDir, stdout, stderr, nil
}

func createFileFor(path string, uid, gid int) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}

	err = os.Chown(path, uid, gid)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (a *RuncAdapter) BuildSpec(
	systemRoot string,
	jobName string,
	cfg *bpm.Config,
	user specs.User,
) (specs.Spec, error) {
	process := &specs.Process{
		User:            user,
		Args:            append([]string{cfg.Executable}, cfg.Args...),
		Env:             cfg.Env,
		Cwd:             "/",
		Rlimits:         []specs.LinuxRlimit{},
		NoNewPrivileges: true,
	}

	mounts := defaultMounts()
	mounts = append(mounts, boshMounts(systemRoot, jobName, cfg.Name)...)
	mounts = append(mounts, systemIdentityMounts()...)

	var resources *specs.LinuxResources
	if cfg.Limits != nil {
		if cfg.Limits.Memory != nil {
			memLimit, err := bytefmt.ToBytes(*cfg.Limits.Memory)
			if err != nil {
				return specs.Spec{}, err
			}

			resources = &specs.LinuxResources{
				Memory: &specs.LinuxMemory{
					Limit: &memLimit,
					Swap:  &memLimit,
				},
			}
		}

		if cfg.Limits.OpenFiles != nil {
			process.Rlimits = append(process.Rlimits, specs.LinuxRlimit{
				Type: "RLIMIT_NOFILE",
				Hard: uint64(*cfg.Limits.OpenFiles),
				Soft: uint64(*cfg.Limits.OpenFiles),
			})
		}
	}

	return specs.Spec{
		Version: specs.Version,
		Platform: specs.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		Process: process,
		Root: specs.Root{
			Path: filepath.Join(bpm.BundlesRoot(), jobName, cfg.Name, "rootfs"),
		},
		Hostname: jobName,
		Mounts:   mounts,
		Linux: &specs.Linux{
			MaskedPaths: []string{
				"/etc/sv",
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
			},
			Namespaces: []specs.LinuxNamespace{
				{Type: "uts"},
				{Type: "mount"},
				{Type: "pid"},
			},
			ReadonlyPaths: []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
			Resources:         resources,
			RootfsPropagation: "private",
		},
	}, nil
}

func boshMounts(systemRoot, jobName, procName string) []specs.Mount {
	return []specs.Mount{
		{
			Destination: filepath.Join(systemRoot, "data", jobName, procName),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "data", jobName, procName),
			Options:     []string{"rbind", "rw"},
		},
		{
			Destination: filepath.Join(systemRoot, "data", "packages"),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "data", "packages"),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: filepath.Join(systemRoot, "jobs", jobName),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "jobs", jobName),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: filepath.Join(systemRoot, "packages"),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "packages"),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: filepath.Join(systemRoot, "sys", "log", jobName),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "sys", "log", jobName),
			Options:     []string{"rbind", "rw"},
		},
	}
}

func defaultMounts() []specs.Mount {
	return []specs.Mount{
		{
			Destination: "/proc",
			Type:        "proc",
			Source:      "proc",
			Options:     nil,
		},
		{
			Destination: "/dev",
			Type:        "tmpfs",
			Source:      "tmpfs",
			Options:     []string{"nosuid", "noexec", "mode=755", "size=65536k"},
		},
		{
			Destination: "/dev/pts",
			Type:        "devpts",
			Source:      "devpts",
			Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"},
		},
		{
			Destination: "/dev/shm",
			Type:        "tmpfs",
			Source:      "shm",
			Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
		},
		{
			Destination: "/dev/mqueue",
			Type:        "mqueue",
			Source:      "mqueue",
			Options:     []string{"nosuid", "noexec", "nodev"},
		},
		{
			Destination: "/sys",
			Type:        "sysfs",
			Source:      "sysfs",
			Options:     []string{"nosuid", "noexec", "nodev", "ro"},
		},
	}
}

func systemIdentityMounts() []specs.Mount {
	return []specs.Mount{
		{
			Destination: "/bin",
			Type:        "bind",
			Source:      "/bin",
			Options:     []string{"nosuid", "nodev", "rbind", "ro"},
		},
		{
			Destination: "/etc",
			Type:        "bind",
			Source:      "/etc",
			Options:     []string{"nosuid", "nodev", "rbind", "ro"},
		},
		{
			Destination: "/usr",
			Type:        "bind",
			Source:      "/usr",
			Options:     []string{"nosuid", "nodev", "rbind", "ro"},
		},
		{
			Destination: "/lib",
			Type:        "bind",
			Source:      "/lib",
			Options:     []string{"nosuid", "nodev", "rbind", "ro"},
		},
		{
			Destination: "/lib64",
			Type:        "bind",
			Source:      "/lib64",
			Options:     []string{"nosuid", "nodev", "rbind", "ro"},
		},
	}
}
