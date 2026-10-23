// Copyright (C) 2026-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package safeio

import (
	"fmt"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// MkdirChown prepares a directory without resolving service-controlled symlinks.
func MkdirChown(path string, uid, gid int) error { return prepareVolume(path, uid, gid, true, false) }

// PrepareVolume preserves support for existing files and creates missing
// directories. Directory modes and ownership are changed through descriptors.
func PrepareVolume(path string, uid, gid int) error {
	return prepareVolume(path, uid, gid, false, true)
}

func prepareVolume(path string, uid, gid int, directoryOnly, chmodExisting bool) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("volume path must be absolute and clean: %s", path)
	}
	fd, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(fd) }() //nolint:errcheck
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
	createdLeaf := false
	for i, name := range parts {
		if name == "" {
			return fmt.Errorf("refusing root volume path")
		}
		leaf := i == len(parts)-1
		flags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_NONBLOCK | unix.O_CLOEXEC
		if !leaf || directoryOnly {
			flags |= unix.O_DIRECTORY
		}
		next, openErr := unix.Openat(fd, name, flags, 0)
		if openErr == unix.ENOENT {
			if err := unix.Mkdirat(fd, name, 0700); err != nil && err != unix.EEXIST {
				return err
			}
			next, openErr = unix.Openat(fd, name, flags|unix.O_DIRECTORY, 0)
			createdLeaf = leaf
		}
		if openErr != nil && !leaf {
			// BOSH owns root-only layout symlinks such as /var/vcap/sys. Only
			// privileged administrators can replace entries in these parents.
			var parent, entry unix.Stat_t
			if unix.Fstat(fd, &parent) == nil && parent.Uid == 0 && parent.Mode&0022 == 0 && unix.Fstatat(fd, name, &entry, unix.AT_SYMLINK_NOFOLLOW) == nil && entry.Mode&unix.S_IFMT == unix.S_IFLNK {
				next, openErr = unix.Openat(fd, name, flags&^unix.O_NOFOLLOW, 0)
			}
		}
		if openErr != nil {
			return fmt.Errorf("opening volume component %s: %w", name, openErr)
		}
		_ = unix.Close(fd) //nolint:errcheck
		fd = next
	}
	var info unix.Stat_t
	if err := unix.Fstat(fd, &info); err != nil {
		return err
	}
	kind := info.Mode & unix.S_IFMT
	if kind != unix.S_IFDIR && kind != unix.S_IFREG {
		return fmt.Errorf("unsupported volume file type: %s", path)
	}
	if kind == unix.S_IFREG && info.Nlink > 1 {
		return fmt.Errorf("refusing multiply linked volume file: %s", path)
	}
	if kind == unix.S_IFDIR && (createdLeaf || chmodExisting) {
		if err := unix.Fchmod(fd, 0700); err != nil {
			return err
		}
	}
	if err := unix.Fchown(fd, uid, gid); err != nil {
		return fmt.Errorf("owning volume %s: %w", path, err)
	}
	return nil
}
