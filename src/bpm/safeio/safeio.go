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
// (e.g. job log directories). The helpers use O_NOFOLLOW + fchown so that a
// pre-planted symlink at the leaf path cannot redirect the operation onto an
// arbitrary host file.
package safeio

import (
	"errors"
	"fmt"
	"os"

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
