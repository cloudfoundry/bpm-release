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
	"bpm/config"
	"fmt"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/bytefmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const ResolvConfDir string = "/run/resolvconf"

type RuncAdapter struct{}

func NewRuncAdapter() *RuncAdapter {
	return &RuncAdapter{}
}

func (a *RuncAdapter) CreateJobPrerequisites(
	bpmCfg *config.BPMConfig,
	procCfg *config.ProcessConfig,
	user specs.User,
) (*os.File, *os.File, error) {
	err := os.MkdirAll(bpmCfg.PidDir(), 0700)
	if err != nil {
		return nil, nil, err
	}

	mountStore, err := checkDirExists(filepath.Dir(bpmCfg.StoreDir()))
	if err != nil {
		return nil, nil, err
	}

	directories := append(
		procCfg.Volumes,
		bpmCfg.DataDir(),
		bpmCfg.LogDir(),
		bpmCfg.TempDir(),
	)

	if mountStore {
		directories = append(directories, bpmCfg.StoreDir())
	}

	for _, job := range procCfg.AdditionalJobs {
		additionalBPMCfg := config.NewBPMConfig(config.BoshRoot(), job, job)
		directories = append(
			directories,
			additionalBPMCfg.DataDir(),
			additionalBPMCfg.LogDir(),
			additionalBPMCfg.TempDir(),
		)

		if mountStore {
			directories = append(directories, additionalBPMCfg.StoreDir())
		}
	}

	for _, dir := range directories {
		err = createDirFor(dir, int(user.UID), int(user.GID))
		if err != nil {
			return nil, nil, err
		}
	}

	files := make([]*os.File, 2)
	paths := []string{bpmCfg.Stdout(), bpmCfg.Stderr()}
	for i, path := range paths {
		f, err := createFileFor(path, int(user.UID), int(user.GID))
		if err != nil {
			return nil, nil, err
		}
		files[i] = f
	}

	return files[0], files[1], nil
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
	bpmCfg *config.BPMConfig,
	procCfg *config.ProcessConfig,
	user specs.User,
) (specs.Spec, error) {
	process := &specs.Process{
		User:            user,
		Args:            append([]string{procCfg.Executable}, procCfg.Args...),
		Env:             processEnvironment(procCfg.Env, bpmCfg),
		Cwd:             "/",
		Rlimits:         []specs.POSIXRlimit{},
		NoNewPrivileges: true,
	}

	mountStore, err := checkDirExists(filepath.Dir(bpmCfg.StoreDir()))
	if err != nil {
		return specs.Spec{}, err
	}

	mountResolvConf, err := checkDirExists(ResolvConfDir)
	if err != nil {
		return specs.Spec{}, err
	}

	mounts := requiredMounts()
	mounts = append(mounts, systemIdentityMounts(mountResolvConf)...)
	mounts = append(mounts, boshMounts(bpmCfg, mountStore)...)
	mounts = append(mounts, userProvidedIdentityMounts(procCfg.Volumes)...)
	mounts = append(mounts, additionalConfigMounts(procCfg, mountStore)...)

	var resources *specs.LinuxResources
	if procCfg.Limits != nil {
		resources = &specs.LinuxResources{}

		if procCfg.Limits.Memory != nil {
			memLimit, err := bytefmt.ToBytes(*procCfg.Limits.Memory)
			if err != nil {
				return specs.Spec{}, err
			}

			signedMemLimit := int64(memLimit)
			resources.Memory = &specs.LinuxMemory{
				Limit: &signedMemLimit,
				Swap:  &signedMemLimit,
			}
		}

		if procCfg.Limits.Processes != nil {
			resources.Pids = &specs.LinuxPids{
				Limit: *procCfg.Limits.Processes,
			}
		}

		if procCfg.Limits.OpenFiles != nil {
			process.Rlimits = append(process.Rlimits, specs.POSIXRlimit{
				Type: "RLIMIT_NOFILE",
				Hard: uint64(*procCfg.Limits.OpenFiles),
				Soft: uint64(*procCfg.Limits.OpenFiles),
			})
		}
	}

	return specs.Spec{
		Version: specs.Version,
		Process: process,
		Root: &specs.Root{
			Path: bpmCfg.RootFSPath(),
		},
		Mounts: mounts,
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
			Seccomp: seccomp,
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

func requiredMounts() []specs.Mount {
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

func systemIdentityMounts(mountResolvConf bool) []specs.Mount {
	mounts := []specs.Mount{
		identityBindMountWithOptions("/bin", "nosuid", "nodev", "rbind", "ro"),
		identityBindMountWithOptions("/usr", "nosuid", "nodev", "rbind", "ro"),
		identityBindMountWithOptions("/etc", "nosuid", "nodev", "rbind", "ro"),
		identityBindMountWithOptions("/lib", "nosuid", "nodev", "rbind", "ro"),
		identityBindMountWithOptions("/lib64", "nosuid", "nodev", "rbind", "ro"),
	}

	if mountResolvConf {
		mounts = append(mounts, identityBindMountWithOptions("/run/resolvconf", "nodev", "nosuid", "noexec", "rbind", "ro"))
	}

	return mounts
}

func boshMounts(bpmCfg *config.BPMConfig, mountStore bool) []specs.Mount {
	mounts := []specs.Mount{
		identityBindMountWithOptions(bpmCfg.DataDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		identityBindMountWithOptions(bpmCfg.LogDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		identityBindMountWithOptions(bpmCfg.DataPackageDir(), "nodev", "nosuid", "rbind", "ro"),
		identityBindMountWithOptions(bpmCfg.JobDir(), "nodev", "nosuid", "rbind", "ro"),
		identityBindMountWithOptions(bpmCfg.PackageDir(), "nodev", "nosuid", "rbind", "ro"),
		bindMountWithOptions("/var/tmp", bpmCfg.TempDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		bindMountWithOptions("/tmp", bpmCfg.TempDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
	}

	if mountStore {
		mounts = append(mounts, identityBindMountWithOptions(bpmCfg.StoreDir(), "nodev", "nosuid", "noexec", "rbind", "rw"))
	}

	return mounts
}

func userProvidedIdentityMounts(volumes []string) []specs.Mount {
	var mnts []specs.Mount

	for _, vol := range volumes {
		mnts = append(mnts, identityBindMountWithOptions(vol, "nodev", "nosuid", "noexec", "rbind", "rw"))
	}

	return mnts
}

func additionalConfigMounts(procCfg *config.ProcessConfig, mountStore bool) []specs.Mount {
	var mnts []specs.Mount

	for _, job := range procCfg.AdditionalJobs {
		additionalBPMCfg := config.NewBPMConfig(config.BoshRoot(), job, job)
		mnts = append(mnts, boshMounts(additionalBPMCfg, mountStore)...)
	}

	return mnts
}

func identityBindMountWithOptions(path string, options ...string) specs.Mount {
	return bindMountWithOptions(path, path, options...)
}

func bindMountWithOptions(dest, src string, options ...string) specs.Mount {
	return specs.Mount{
		Destination: dest,
		Type:        "bind",
		Source:      src,
		Options:     options,
	}
}

func processEnvironment(env []string, cfg *config.BPMConfig) []string {
	return append(
		env,
		fmt.Sprintf("TMPDIR=%s", cfg.TempDir()),
		fmt.Sprintf("BPM_ID=%s", cfg.ContainerID()),
	)
}

func checkDirExists(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	return false, nil
}
