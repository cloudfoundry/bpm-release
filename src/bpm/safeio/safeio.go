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

// Package safeio provides helpers for performing privileged file operations
// against paths that live in directories writable by less-privileged users
// (e.g. job log directories). The helpers use O_NOFOLLOW, Lstat, and lchown
// so that a pre-planted symlink cannot redirect the operation onto an
// arbitrary host file. MkdirAllChown, Lchown, and Lchmod walk and check every
// path component to ensure symlinks cannot redirect privileged operations;
// OpenAppendChown operates on an existing parent and guards the leaf.
package safeio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// OpenAppendChown opens path for read/write/create/append with O_NOFOLLOW so
// that a symlink at the leaf path will not be followed, and then changes the
// ownership of the resulting file via the open file descriptor (fchown(2)).
//
// If the leaf component of path is a symlink the open fails with ELOOP (Linux
// returns ELOOP for O_NOFOLLOW on a symlink leaf) and the chown is never
// attempted.
//
// On success the caller owns the returned *os.File and is responsible for
// closing it.
func OpenAppendChown(path string, uid, gid int, perm os.FileMode) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND|unix.O_NOFOLLOW, perm)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return nil, fmt.Errorf("refusing to open symlink at %s: %w", path, err)
		}
		return nil, err
	}

	if err := f.Chown(uid, gid); err != nil {
		_ = f.Close() //nolint:errcheck
		return nil, fmt.Errorf("chown %s: %w", path, err)
	}

	return f, nil
}

// trustedSymlinkComponent reports whether a symlink found at an intermediate
// path component may safely be followed. parent is an open descriptor for the
// directory containing the symlink, and linkStat is the lstat of the link.
//
// The threat model is a compromised container process running as an
// unprivileged user (vcap) that can create and replace entries in directories
// that user owns. Such a process can neither create a root-owned symlink nor
// replace one that sits in a directory only root can write to. A symlink is
// therefore safe to follow exactly when the link itself is root-owned and its
// parent directory is root-owned and not group- or other-writable.
//
// This is a structural property rather than a hardcoded path allowlist on
// purpose. BOSH roots are configurable (see bosh.Path, which exists so several
// roots can coexist on one machine), and which of sys, sys/log or sys/run is a
// symlink varies by stemcell and release — syslog-release documents sys/log as
// a symlink into /var/vcap/data. Keying off ownership covers every such layout,
// and unlike an absolute literal it stays exercisable under a test root.
func trustedSymlinkComponent(parent int, linkStat *unix.Stat_t) bool {
	if linkStat.Uid != 0 {
		return false
	}

	var parentStat unix.Stat_t
	if err := unix.Fstat(parent, &parentStat); err != nil {
		return false
	}

	return parentStat.Uid == 0 && parentStat.Mode&(unix.S_IWGRP|unix.S_IWOTH) == 0
}

// MkdirAllChown creates a directory (and any parent directories) with 0700
// permissions and chowns the leaf directory to the specified uid and gid.
//
// The path is walked one component at a time from the filesystem root, and
// every component is opened with O_NOFOLLOW. A symlink anywhere in the path —
// not just at the leaf — is refused. Checking only the leaf is not sufficient:
// an attacker who can plant a symlink at an intermediate component redirects
// the whole subtree, so MkdirAll would create the leaf under the symlink's
// target and the subsequent chown would hand that target to uid/gid.
//
// The final chown is issued against the open directory descriptor (fchown(2)),
// so the path cannot be swapped between the walk and the ownership change.
//
// A symlink at an intermediate component is followed only when it satisfies
// trustedSymlinkComponent, which the standard BOSH sys/log and sys/run symlinks
// do. A symlink at the leaf is always refused.
func MkdirAllChown(path string, uid, gid int) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing to create directory at relative path %s", path)
	}

	// O_PATH would be enough to walk, but it makes O_NOFOLLOW return a
	// descriptor to the symlink itself rather than ELOOP, so use a real
	// directory open instead.
	parent, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open /: %w", err)
	}
	defer func() { _ = unix.Close(parent) }() //nolint:errcheck

	components := strings.Split(strings.Trim(filepath.Clean(path), "/"), "/")
	for i, component := range components {
		isLeaf := i == len(components)-1
		if err := unix.Mkdirat(parent, component, 0700); err != nil && !errors.Is(err, unix.EEXIST) {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}

		// Classify the component before opening it. Errno alone is not
		// portable here: with O_DIRECTORY set, a symlink reports ENOTDIR on
		// some platforms and ELOOP on others.
		var st unix.Stat_t
		if err := unix.Fstatat(parent, component, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}
		openFlags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFLNK:
			if !isLeaf && trustedSymlinkComponent(parent, &st) {
				break
			}
			return fmt.Errorf("refusing to create directory through symlink at %s", path)
		case unix.S_IFDIR:
			openFlags |= unix.O_NOFOLLOW
		default:
			return fmt.Errorf("path %s exists and is not a directory", path)
		}

		child, err := unix.Openat(parent, component, openFlags, 0)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}

		_ = unix.Close(parent) //nolint:errcheck
		parent = child
	}

	return unix.Fchown(parent, uid, gid)
}

// Lchown changes the ownership of the named file or directory without following
// symbolic links. It walks every path component, refusing a symlink at the leaf
// outright and following one at an intermediate component only when
// trustedSymlinkComponent allows it.
func Lchown(path string, uid, gid int) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing to chown relative path %s", path)
	}

	cleanPath := filepath.Clean(path)
	components := strings.Split(strings.Trim(cleanPath, "/"), "/")
	if len(components) == 0 {
		return errors.New("cannot chown root")
	}

	parent, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open /: %w", err)
	}
	defer func() { _ = unix.Close(parent) }() //nolint:errcheck

	var currentPath string
	for i, component := range components {
		currentPath = filepath.Join(currentPath, component)
		if !strings.HasPrefix(currentPath, "/") {
			currentPath = "/" + currentPath
		}

		var st unix.Stat_t
		if err := unix.Fstatat(parent, component, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		isLeaf := (i == len(components)-1)
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFLNK:
			if !isLeaf && trustedSymlinkComponent(parent, &st) {
				child, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
				if err != nil {
					return fmt.Errorf("open %s: %w", path, err)
				}
				_ = unix.Close(parent) //nolint:errcheck
				parent = child
				continue
			}
			return fmt.Errorf("refusing to chown symlink at %s", path)
		case unix.S_IFDIR:
			if isLeaf {
				// Leaf directory: open with O_DIRECTORY|O_NOFOLLOW and fchown
				child, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
				if err != nil {
					return fmt.Errorf("open %s: %w", path, err)
				}
				defer func() { _ = unix.Close(child) }() //nolint:errcheck
				return unix.Fchown(child, uid, gid)
			}
			child, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			if err != nil {
				return fmt.Errorf("open %s: %w", path, err)
			}
			_ = unix.Close(parent) //nolint:errcheck
			parent = child
		default:
			// Non-directory (regular file, etc.)
			if isLeaf {
				return unix.Fchownat(parent, component, uid, gid, unix.AT_SYMLINK_NOFOLLOW)
			}
			return fmt.Errorf("path component %s exists and is not a directory", currentPath)
		}
	}

	return nil
}

// Lchmod changes the mode of the named file or directory without following
// symbolic links. It walks every path component, refusing a symlink at the leaf
// outright and following one at an intermediate component only when
// trustedSymlinkComponent allows it.
func Lchmod(path string, mode uint32) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("refusing to chmod relative path %s", path)
	}

	cleanPath := filepath.Clean(path)
	components := strings.Split(strings.Trim(cleanPath, "/"), "/")
	if len(components) == 0 {
		return errors.New("cannot chmod root")
	}

	parent, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open /: %w", err)
	}
	defer func() { _ = unix.Close(parent) }() //nolint:errcheck

	var currentPath string
	for i, component := range components {
		currentPath = filepath.Join(currentPath, component)
		if !strings.HasPrefix(currentPath, "/") {
			currentPath = "/" + currentPath
		}

		var st unix.Stat_t
		if err := unix.Fstatat(parent, component, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		isLeaf := (i == len(components)-1)
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFLNK:
			if !isLeaf && trustedSymlinkComponent(parent, &st) {
				child, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
				if err != nil {
					return fmt.Errorf("open %s: %w", path, err)
				}
				_ = unix.Close(parent) //nolint:errcheck
				parent = child
				continue
			}
			return fmt.Errorf("refusing to chmod symlink at %s", path)
		case unix.S_IFDIR:
			if isLeaf {
				child, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
				if err != nil {
					return fmt.Errorf("open %s: %w", path, err)
				}
				defer func() { _ = unix.Close(child) }() //nolint:errcheck
				return unix.Fchmod(child, mode)
			}
			child, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
			if err != nil {
				return fmt.Errorf("open %s: %w", path, err)
			}
			_ = unix.Close(parent) //nolint:errcheck
			parent = child
		default:
			if isLeaf {
				child, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
				if err != nil {
					return fmt.Errorf("open %s: %w", path, err)
				}
				defer func() { _ = unix.Close(child) }() //nolint:errcheck
				return unix.Fchmod(child, mode)
			}
			return fmt.Errorf("path component %s exists and is not a directory", currentPath)
		}
	}

	return nil
}
