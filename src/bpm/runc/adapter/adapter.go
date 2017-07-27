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
	systemRoot, jobName, procName string,
	cfg *bpm.Config,
	user specs.User,
) (string, *os.File, *os.File, error) {
	builder := newFilePathBuilder(systemRoot, jobName, procName)

	err := os.MkdirAll(builder.pidDir(), 0700)
	if err != nil {
		return "", nil, nil, err
	}

	err = os.MkdirAll(builder.logDir(), 0750)
	if err != nil {
		return "", nil, nil, err
	}
	err = os.Chown(builder.logDir(), int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	stdout, err := createFileFor(builder.stdout(), int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	stderr, err := createFileFor(builder.stderr(), int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	err = createDirFor(builder.dataDir(), int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	err = createDirFor(builder.tempDir(), int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	for _, vol := range cfg.Volumes {
		err := createDirFor(vol, int(user.UID), int(user.GID))
		if err != nil {
			return "", nil, nil, err
		}
	}

	return builder.pidDir(), stdout, stderr, nil
}

func createDirFor(path string, uid, gid int) error {
	err := os.MkdirAll(path, 0700)
	if err != nil {
		return err
	}

	return os.Chown(path, uid, gid)
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
	systemRoot, jobName, procName string,
	cfg *bpm.Config,
	user specs.User,
) (specs.Spec, error) {
	builder := newFilePathBuilder(systemRoot, jobName, procName)

	process := &specs.Process{
		User:            user,
		Args:            append([]string{cfg.Executable}, cfg.Args...),
		Env:             processEnvironment(cfg.Env, builder.tempDir()),
		Cwd:             "/",
		Rlimits:         []specs.LinuxRlimit{},
		NoNewPrivileges: true,
	}

	mounts := defaultMounts()
	mounts = append(mounts, boshMounts(builder)...)
	mounts = append(mounts, systemIdentityMounts()...)
	mounts = append(mounts, userProvidedIdentityMounts(cfg.Volumes)...)

	var resources *specs.LinuxResources
	if cfg.Limits != nil {
		resources = &specs.LinuxResources{}

		if cfg.Limits.Memory != nil {
			memLimit, err := bytefmt.ToBytes(*cfg.Limits.Memory)
			if err != nil {
				return specs.Spec{}, err
			}

			resources.Memory = &specs.LinuxMemory{
				Limit: &memLimit,
				Swap:  &memLimit,
			}
		}

		if cfg.Limits.Processes != nil {
			resources.Pids = &specs.LinuxPids{
				Limit: *cfg.Limits.Processes,
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
			Path: filepath.Join(bpm.BundlesRoot(), jobName, procName, "rootfs"),
		},
		Hostname: jobName,
		Mounts:   mounts,
		Linux: &specs.Linux{
			MaskedPaths: []string{
				"/etc/sv",
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/sched_debug",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/sys/firmware",
			},
			Namespaces: []specs.LinuxNamespace{
				{Type: "ipc"},
				{Type: "mount"},
				{Type: "pid"},
				{Type: "uts"},
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

func boshMounts(builder filePathBuilder) []specs.Mount {
	return []specs.Mount{
		{
			Destination: builder.dataDir(),
			Type:        "bind",
			Source:      builder.dataDir(),
			Options:     []string{"rbind", "rw"},
		},
		{
			Destination: "/tmp",
			Type:        "bind",
			Source:      builder.tempDir(),
			Options:     []string{"rbind", "rw"},
		},
		{
			Destination: builder.dataPackageDir(),
			Type:        "bind",
			Source:      builder.dataPackageDir(),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: builder.jobDir(),
			Type:        "bind",
			Source:      builder.jobDir(),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: builder.packageDir(),
			Type:        "bind",
			Source:      builder.packageDir(),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: builder.logDir(),
			Type:        "bind",
			Source:      builder.logDir(),
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

func userProvidedIdentityMounts(volumes []string) []specs.Mount {
	var mnts []specs.Mount

	for _, vol := range volumes {
		mnts = append(mnts, specs.Mount{
			Destination: vol,
			Type:        "bind",
			Source:      vol,
			Options:     []string{"rbind", "rw"},
		})
	}

	return mnts
}

func processEnvironment(env []string, tmpDir string) []string {
	return append(env, fmt.Sprintf("TMPDIR=%s", tmpDir))
}

type filePathBuilder struct {
	systemRoot string
	jobName    string
	procName   string
}

func newFilePathBuilder(systemRoot, jobName, procName string) filePathBuilder {
	return filePathBuilder{
		systemRoot: systemRoot,
		jobName:    jobName,
		procName:   procName,
	}
}

func (b filePathBuilder) dataDir() string {
	return filepath.Join(b.systemRoot, "data", b.jobName)
}

func (b filePathBuilder) tempDir() string {
	return filepath.Join(b.dataDir(), "tmp")
}

func (b filePathBuilder) logDir() string {
	return filepath.Join(b.systemRoot, "sys", "log", b.jobName)
}

func (b filePathBuilder) stdout() string {
	return filepath.Join(b.logDir(), fmt.Sprintf("%s.out.log", b.procName))
}

func (b filePathBuilder) stderr() string {
	return filepath.Join(b.logDir(), fmt.Sprintf("%s.err.log", b.procName))
}

func (b filePathBuilder) pidDir() string {
	return filepath.Join(b.systemRoot, "sys", "run", "bpm", b.jobName)
}

func (b filePathBuilder) packageDir() string {
	return filepath.Join(b.systemRoot, "packages")
}

func (b filePathBuilder) dataPackageDir() string {
	return filepath.Join(b.systemRoot, "data", "packages")
}

func (b filePathBuilder) jobDir() string {
	return filepath.Join(b.systemRoot, "jobs", b.jobName)
}
