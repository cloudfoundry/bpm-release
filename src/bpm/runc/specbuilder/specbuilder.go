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

package specbuilder

import (
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func DefaultSpec() *specs.Spec {
	return &specs.Spec{
		Version: specs.Version,
		Process: &specs.Process{
			Capabilities:    &specs.LinuxCapabilities{},
			NoNewPrivileges: true,
		},
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
			ReadonlyPaths: []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
			Resources:         &specs.LinuxResources{},
			RootfsPropagation: "private",
			Seccomp:           DefaultSeccomp(),
		},
		Mounts: []specs.Mount{
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
		},
	}
}

type SpecOption func(*specs.Spec)

func Build(opts ...SpecOption) *specs.Spec {
	spec := DefaultSpec()
	Apply(spec, opts...)
	return spec
}

func Apply(spec *specs.Spec, opts ...SpecOption) {
	for _, opt := range opts {
		opt(spec)
	}
}

func WithRootFilesystem(path string) SpecOption {
	return func(spec *specs.Spec) {
		spec.Root = &specs.Root{
			Path: path,
		}
	}
}

func WithNamespace(namespace specs.LinuxNamespaceType) SpecOption {
	return func(spec *specs.Spec) {
		spec.Linux.Namespaces = append(spec.Linux.Namespaces, specs.LinuxNamespace{Type: namespace})
	}
}

func WithUser(user specs.User) SpecOption {
	return func(spec *specs.Spec) {
		spec.Process.User = user
	}
}

func WithProcess(
	executable string,
	args []string,
	environment []string,
	cwd string,
) SpecOption {
	return func(spec *specs.Spec) {
		spec.Process.Args = append([]string{executable}, args...)
		spec.Process.Env = environment
		spec.Process.Cwd = cwd
	}
}

func WithCapabilities(capabilities []string) SpecOption {
	// We do not set the Effective set, as it is computed dynamically by the
	// kernel from the other sets.
	return func(spec *specs.Spec) {
		spec.Process.Capabilities.Ambient = append(spec.Process.Capabilities.Ambient, capabilities...)
		spec.Process.Capabilities.Bounding = append(spec.Process.Capabilities.Bounding, capabilities...)
		spec.Process.Capabilities.Inheritable = append(spec.Process.Capabilities.Inheritable, capabilities...)
		spec.Process.Capabilities.Permitted = append(spec.Process.Capabilities.Permitted, capabilities...)
	}
}

func WithMounts(mounts []specs.Mount) SpecOption {
	return func(spec *specs.Spec) {
		spec.Mounts = append(spec.Mounts, mounts...)
	}
}

func WithMemoryLimit(limit int64) SpecOption {
	return func(spec *specs.Spec) {
		spec.Linux.Resources.Memory = &specs.LinuxMemory{
			Limit: &limit,
			Swap:  &limit,
		}
	}
}

func WithPidLimit(limit int64) SpecOption {
	return func(spec *specs.Spec) {
		spec.Linux.Resources.Pids = &specs.LinuxPids{
			Limit: limit,
		}
	}
}

func WithOpenFileLimit(limit uint64) SpecOption {
	return func(spec *specs.Spec) {
		spec.Process.Rlimits = append(spec.Process.Rlimits, specs.POSIXRlimit{
			Type: "RLIMIT_NOFILE",
			Hard: limit,
			Soft: limit,
		})
	}
}

var RootUser = specs.User{
	UID: 0,
	GID: 0,
}

func WithPrivileged() SpecOption {
	return func(spec *specs.Spec) {
		Apply(spec, WithCapabilities(DefaultPrivilegedCapabilities()))
		Apply(spec, WithUser(RootUser))

		spec.Process.NoNewPrivileges = false

		spec.Linux.MaskedPaths = []string{}
		spec.Linux.ReadonlyPaths = []string{}
		spec.Linux.Seccomp = nil

		for i, mount := range spec.Mounts {
			spec.Mounts[i].Options = removeNosuidMountOption(mount.Options)
		}
	}
}

func removeNosuidMountOption(opts []string) []string {
	for i := 0; i < len(opts); i++ {
		if opts[i] == "nosuid" {
			return append(opts[:i], opts[i+1:]...)
		}
	}

	return opts
}
