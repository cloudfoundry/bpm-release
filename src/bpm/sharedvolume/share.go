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

package sharedvolume

import (
	"github.com/opencontainers/runc/libcontainer/mount"
	"golang.org/x/sys/unix"
)

// MakeShared takes a path and turns it into a shared mountpoint. Due to the
// requirement of only being able to share an existing mountpoint this function
// will also bind mount a path to itself if it is not already a mountpoint. In
// the case of an error in making the mountpoint shared this identity mount
// will be rolled back.
func MakeShared(path string) error {
	isMount, err := mount.Mounted(path)
	if err != nil {
		return err
	}

	if !isMount {
		if err := unix.Mount(path, path, "", unix.MS_BIND, ""); err != nil {
			return err
		}
	}

	// These options were taken from tracing the execution of `mount
	// --make-shared`.
	if err := unix.Mount("none", path, "", unix.MS_SHARED, ""); err != nil {
		if !isMount {
			_ = unix.Unmount(path, 0)
			return err
		}
	}

	return nil
}
