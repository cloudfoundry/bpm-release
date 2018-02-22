// Copyright (C) 2017-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/bytefmt"
	"code.cloudfoundry.org/lager"

	"github.com/opencontainers/runtime-spec/specs-go"
)

const (
	ResolvConfDir string = "/run/resolvconf"
	DefaultLang   string = "en_US.UTF-8"
)

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

	var directories []string
	for _, vol := range procCfg.AdditionalVolumes {
		directories = append(directories, vol.Path)
	}

	directories = append(
		directories,
		bpmCfg.LogDir(),
		bpmCfg.TempDir(),
	)

	if procCfg.EphemeralDisk {
		directories = append(directories, bpmCfg.DataDir())
	}

	if procCfg.PersistentDisk {
		storeExists, err := checkDirExists(filepath.Dir(bpmCfg.StoreDir()))
		if err != nil {
			return nil, nil, err
		}

		if !storeExists {
			return nil, nil, errors.New("requested persistent disk does not exist")
		}

		directories = append(directories, bpmCfg.StoreDir())
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
	logger lager.Logger,
	bpmCfg *config.BPMConfig,
	procCfg *config.ProcessConfig,
	user specs.User,
) (specs.Spec, error) {
	cwd := bpmCfg.JobDir()
	if procCfg.WorkDir != "" {
		cwd = procCfg.WorkDir
	}

	process := &specs.Process{
		User:            user,
		Args:            append([]string{procCfg.Executable}, procCfg.Args...),
		Env:             processEnvironment(procCfg.Env, bpmCfg),
		Cwd:             cwd,
		Rlimits:         []specs.POSIXRlimit{},
		NoNewPrivileges: true,
		Capabilities:    processCapabilities(procCfg.Capabilities),
	}

	mountResolvConf, err := checkDirExists(ResolvConfDir)
	if err != nil {
		return specs.Spec{}, err
	}

	mounts := requiredMounts()
	mounts = append(mounts, systemIdentityMounts(mountResolvConf)...)
	mounts = append(mounts, boshMounts(bpmCfg, procCfg.EphemeralDisk, procCfg.PersistentDisk)...)
	mounts = append(mounts, userProvidedIdentityMounts(logger, bpmCfg, procCfg.AdditionalVolumes)...)

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
		identityBindMountWithOptions("/bin", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/usr", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/etc", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/lib", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/lib64", "nosuid", "nodev", "bind", "ro"),
	}

	if mountResolvConf {
		mounts = append(mounts, identityBindMountWithOptions("/run/resolvconf", "nodev", "nosuid", "noexec", "bind", "ro"))
	}

	return mounts
}

func boshMounts(bpmCfg *config.BPMConfig, mountData, mountStore bool) []specs.Mount {
	mounts := []specs.Mount{
		identityBindMountWithOptions(bpmCfg.DataPackageDir(), "nodev", "nosuid", "bind", "ro"),
		identityBindMountWithOptions(bpmCfg.JobDir(), "nodev", "nosuid", "bind", "ro"),
		identityBindMountWithOptions(bpmCfg.LogDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		identityBindMountWithOptions(bpmCfg.PackageDir(), "nodev", "nosuid", "bind", "ro"),
		identityBindMountWithOptions(bpmCfg.TempDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		bindMountWithOptions("/var/tmp", bpmCfg.TempDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		bindMountWithOptions("/tmp", bpmCfg.TempDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
	}

	if mountData {
		mounts = append(mounts, identityBindMountWithOptions(bpmCfg.DataDir(), "nodev", "nosuid", "noexec", "rbind", "rw"))
	}

	if mountStore {
		mounts = append(mounts, identityBindMountWithOptions(bpmCfg.StoreDir(), "nodev", "nosuid", "noexec", "rbind", "rw"))
	}

	return mounts
}

func userProvidedIdentityMounts(logger lager.Logger, bpmCfg *config.BPMConfig, volumes []config.Volume) []specs.Mount {
	var mnts []specs.Mount
	mntsSeen := map[string]bool{
		bpmCfg.DataDir():  true,
		bpmCfg.StoreDir(): true,
	}

	for _, vol := range volumes {
		if _, ok := mntsSeen[vol.Path]; ok {
			logger.Info("duplicate-volume", lager.Data{"volume": vol.Path})
			continue
		}
		execOpt := "noexec"
		if vol.AllowExecutions {
			execOpt = "exec"
		}
		writeOpt := "ro"
		if vol.Writable {
			writeOpt = "rw"
		}
		mnts = append(mnts, identityBindMountWithOptions(vol.Path, "nodev", "nosuid", execOpt, "rbind", writeOpt))
		mntsSeen[vol.Path] = true
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

func processEnvironment(env map[string]string, cfg *config.BPMConfig) []string {
	var environ []string

	for k, v := range env {
		environ = append(environ, fmt.Sprintf("%s=%s", k, v))
	}

	if _, ok := env["TMPDIR"]; !ok {
		environ = append(environ, fmt.Sprintf("TMPDIR=%s", cfg.TempDir()))
	}

	if _, ok := env["LANG"]; !ok {
		environ = append(environ, fmt.Sprintf("LANG=%s", DefaultLang))
	}

	if _, ok := env["PATH"]; !ok {
		environ = append(environ, fmt.Sprintf("PATH=%s", DefaultPath(cfg)))
	}

	if _, ok := env["HOME"]; !ok {
		environ = append(environ, fmt.Sprintf("HOME=%s", cfg.DataDir()))
	}

	return environ
}

// Returns the specs.LinuxCapabilities for a given slice of Capabilities.
// We do not set the Effective set, as it is computed dynamically by the
// kernel from the other sets.
func processCapabilities(caps []string) *specs.LinuxCapabilities {
	capsWithPrefix := make([]string, len(caps))
	for i := 0; i < len(caps); i++ {
		capsWithPrefix[i] = fmt.Sprintf("CAP_%s", caps[i])
	}

	return &specs.LinuxCapabilities{
		Ambient:     capsWithPrefix,
		Bounding:    capsWithPrefix,
		Effective:   []string{},
		Inheritable: capsWithPrefix,
		Permitted:   capsWithPrefix,
	}
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

func DefaultPath(cfg *config.BPMConfig) string {
	defaultPath := "%s:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin:."
	defaultPath = fmt.Sprintf(defaultPath, filepath.Join(cfg.JobDir(), "bin"))

	return defaultPath
}
