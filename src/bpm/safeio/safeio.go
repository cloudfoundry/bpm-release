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

// splitAbsPath splits an absolute path into its components.
func splitAbsPath(path, op string) ([]string, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("refusing to %s at relative path %s", op, path)
	}

	cleanPath := filepath.Clean(path)
	if cleanPath == "/" {
		return nil, fmt.Errorf("refusing to %s at /", op)
	}

	return strings.Split(strings.TrimPrefix(cleanPath, "/"), "/"), nil
}

func openDirComponent(parent int, component string, followTrusted bool, path, refusal string) (int, error) {
	var st unix.Stat_t
	if err := unix.Fstatat(parent, component, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return -1, fmt.Errorf("stat %s: %w", path, err)
	}

	openFlags := unix.O_RDONLY | unix.O_DIRECTORY | unix.O_CLOEXEC
	switch st.Mode & unix.S_IFMT {
	case unix.S_IFLNK:
		if !followTrusted || !trustedSymlinkComponent(parent, &st) {
			return -1, fmt.Errorf("refusing to %s at %s", refusal, path)
		}
	case unix.S_IFDIR:
		openFlags |= unix.O_NOFOLLOW
	default:
		return -1, fmt.Errorf("path component %s of %s exists and is not a directory", component, path)
	}

	child, err := unix.Openat(parent, component, openFlags, 0)
	if err != nil {
		return -1, fmt.Errorf("open %s: %w", path, err)
	}

	return child, nil
}

//openLeafParent walks every component of path except the leaf and returns a
// descriptor for the leaf's parent directory, the leaf name, and the leaf's
// lstat result. The caller must close the descriptor. It refuses a symlink
// leaf and follows an intermediate symlink only when trustedSymlinkComponent
// allows it.
func openLeafParent(path, op string) (int, string, unix.Stat_t, error) {
	var st unix.Stat_t

	components, err := splitAbsPath(path, op)
	if err != nil {
		return -1, "", st, err
	}

	parent, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return -1, "", st, fmt.Errorf("open /: %w", err)
	}

	refusal := op + " symlink"
	for _, component := range components[:len(components)-1] {
		child, err := openDirComponent(parent, component, true, path, refusal)
		_ = unix.Close(parent) //nolint:errcheck
		if err != nil {
			return -1, "", st, err
		}
		parent = child
	}

	leaf := components[len(components)-1]
	if err := unix.Fstatat(parent, leaf, &st, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		_ = unix.Close(parent) //nolint:errcheck
		return -1, "", st, fmt.Errorf("stat %s: %w", path, err)
	}
	if st.Mode&unix.S_IFMT == unix.S_IFLNK {
		_ = unix.Close(parent) //nolint:errcheck
		return -1, "", st, fmt.Errorf("refusing to %s at %s", refusal, path)
	}

	return parent, leaf, st, nil
}

// MkdirAllChown creates a directory (and any parent directories) with 0700
// permissions and chowns the leaf directory to the specified uid and gid.
//
// The path is walked one component at a time from the filesystem root, and
// components are opened with O_NOFOLLOW except for trusted intermediate
// symlinks. Checking only the leaf is not sufficient: an attacker who can
// plant a symlink at an intermediate component redirects the whole subtree,
// so MkdirAll would create the leaf under the symlink's target and the
// subsequent chown would hand that target to uid/gid.
//
// The final chown is issued against the open directory descriptor (fchown(2)),
// so the path cannot be swapped between the walk and the ownership change.
//
// A symlink at an intermediate component is followed only when it satisfies
// trustedSymlinkComponent, which the standard BOSH sys/log and sys/run symlinks
// do. A symlink at the leaf is always refused.
func MkdirAllChown(path string, uid, gid int) error {
	components, err := splitAbsPath(path, "create directory")
	if err != nil {
		return err
	}

	parent, err := unix.Open("/", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open /: %w", err)
	}
	defer func() { _ = unix.Close(parent) }() //nolint:errcheck

	for i, component := range components {
		if err := unix.Mkdirat(parent, component, 0700); err != nil && !errors.Is(err, unix.EEXIST) {
			return fmt.Errorf("mkdir %s: %w", path, err)
		}

		isLeaf := i == len(components)-1
		child, err := openDirComponent(parent, component, !isLeaf, path, "create directory through symlink")
		if err != nil {
			return err
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
	parent, leaf, st, err := openLeafParent(path, "chown")
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parent) }() //nolint:errcheck

	if st.Mode&unix.S_IFMT != unix.S_IFDIR {
		return unix.Fchownat(parent, leaf, uid, gid, unix.AT_SYMLINK_NOFOLLOW)
	}

	child, err := unix.Openat(parent, leaf, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = unix.Close(child) }() //nolint:errcheck

	return unix.Fchown(child, uid, gid)
}

// Lchmod changes the mode of the named file or directory without following
// symbolic links. It walks every path component, refusing a symlink at the leaf
// outright and following one at an intermediate component only when
// trustedSymlinkComponent allows it.
func Lchmod(path string, mode uint32) error {
	parent, leaf, st, err := openLeafParent(path, "chmod")
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(parent) }() //nolint:errcheck

	openFlags := unix.O_RDONLY | unix.O_NOFOLLOW | unix.O_CLOEXEC
	if st.Mode&unix.S_IFMT == unix.S_IFDIR {
		openFlags |= unix.O_DIRECTORY
	} else {
		openFlags |= unix.O_NONBLOCK
	}

	child, err := unix.Openat(parent, leaf, openFlags, 0)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = unix.Close(child) }() //nolint:errcheck

	return unix.Fchmod(child, mode)
}
