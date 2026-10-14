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
// arbitrary host file. MkdirAllChown checks every component of the path;
// OpenAppendChown and Lchown operate on an existing parent and so guard the
// leaf only.
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
// Because every component is checked, a legitimate symlink in the path prefix
// is rejected too. Callers must pass paths whose ancestors are real
// directories.
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

	for _, component := range strings.Split(strings.Trim(filepath.Clean(path), "/"), "/") {
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
		switch st.Mode & unix.S_IFMT {
		case unix.S_IFLNK:
			return fmt.Errorf("refusing to create directory through symlink at %s", path)
		case unix.S_IFDIR:
		default:
			return fmt.Errorf("path %s exists and is not a directory", path)
		}

		// O_NOFOLLOW closes the window between the check above and the open.
		child, err := unix.Openat(parent, component, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}

		_ = unix.Close(parent) //nolint:errcheck
		parent = child
	}

	return unix.Fchown(parent, uid, gid)
}

// Lchown changes the ownership of the named file or directory without following
// symbolic links. If the path is a symlink, it refuses to chown and returns an error.
func Lchown(path string, uid, gid int) error {
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("refusing to chown symlink at %s", path)
	}

	return os.Lchown(path, uid, gid)
}
