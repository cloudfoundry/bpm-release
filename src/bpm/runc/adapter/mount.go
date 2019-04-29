// Copyright (C) 2019-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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
	"sort"
	"strings"

	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

// Mount creates a bind mount description with "to" as the destination and
// "from" as the source. Various MountOptions can be specified to modify the
// mount options. By default, the most restrictive set of mount options are
// used.
func Mount(from, to string, opts ...MountOption) specs.Mount {
	mountOpts := &mountOptions{}

	for _, opt := range opts {
		opt(mountOpts)
	}

	return specs.Mount{
		Destination: to,
		Source:      from,
		Type:        "bind",
		Options:     mountOpts.opts(),
	}
}

// IdentityMount provides a shorthand for Mount which has the same from and to.
func IdentityMount(path string, opts ...MountOption) specs.Mount {
	return Mount(path, path, opts...)
}

// MountOption can be used to alter the mount options to Mount or
// IdentityMount.
type MountOption func(*mountOptions)

// AllowExec allows binaries to be executed from a mount. It maps to the
// exec/noexec mount option.
func AllowExec() MountOption {
	return func(options *mountOptions) {
		options.exec = true
	}
}

// AllowWrites allows writes to a mount. It maps to the ro/rw mount option.
func AllowWrites() MountOption {
	return func(options *mountOptions) {
		options.writable = true
	}
}

// WithRecursiveBind mounts other bind mounts in the source to the destination
// recursively. It maps to the bind/rbind mount option.
func WithRecursiveBind() MountOption {
	return func(options *mountOptions) {
		options.rbind = true
	}
}

type mountOptions struct {
	rbind    bool
	exec     bool
	suid     bool
	dev      bool
	writable bool
}

func (mo mountOptions) opts() []string {
	var opts []string

	if mo.rbind {
		opts = append(opts, "rbind")
	} else {
		opts = append(opts, "bind")
	}

	if !mo.exec {
		opts = append(opts, "noexec")
	}

	if mo.suid {
		opts = append(opts, "suid")
	} else {
		opts = append(opts, "nosuid")
	}

	if !mo.dev {
		opts = append(opts, "nodev")
	}

	if mo.writable {
		opts = append(opts, "rw")
	} else {
		opts = append(opts, "ro")
	}

	return opts
}

type dedupMounts struct {
	set    map[string]specs.Mount
	logger lager.Logger
}

func newMountDedup(logger lager.Logger) *dedupMounts {
	return &dedupMounts{
		set:    make(map[string]specs.Mount),
		logger: logger,
	}
}

func (d *dedupMounts) addMounts(ms []specs.Mount) {
	for _, mount := range ms {
		dst := mount.Destination
		if _, ok := d.set[dst]; ok {
			d.logger.Info("duplicate-mount", lager.Data{"mount": dst})
			continue
		}
		d.set[dst] = mount
	}
}

func (d *dedupMounts) mounts() []specs.Mount {
	ms := make([]specs.Mount, 0, len(d.set))

	for _, mount := range d.set {
		ms = append(ms, mount)
	}

	sort.Slice(ms, func(i, j int) bool {
		iElems := strings.Split(ms[i].Destination, "/")
		jElems := strings.Split(ms[j].Destination, "/")
		return len(iElems) < len(jElems)
	})

	return ms
}
