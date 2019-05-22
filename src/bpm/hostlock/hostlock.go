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

// Package hostlock provides advisory locking capabilities which can be shared
// by any process on the same host (or with a shared of the filesystem). This
// package is currently coupled with BPM concepts but it could be split easily
// if the need ever arose.
package hostlock

import (
	"fmt"
	"path/filepath"

	"bpm/flock"
	"bpm/jobid"
)

// LockedLock represents a lock which has been acquired.
type LockedLock interface {
	// Unlock can be used to unlock and release the lock to let another
	// consumer acquire the lock.
	Unlock() error
}

// Handle represents a namespace of locks which is identified by a particular
// filesystem path.
type Handle struct {
	path string
}

// NewHandle creates a new Handle from a filesystem path. The path must already
// exist.
func NewHandle(path string) *Handle {
	return &Handle{
		path: path,
	}
}

// LockJob places an exclusive advisory lock on a particular BPM job. The
// LockedLock object it returns can be used to release the lock. Subsequent
// calls will block until it is released.
func (h *Handle) LockJob(job, process string) (LockedLock, error) {
	name := jobid.Encode(fmt.Sprintf("%s.%s", job, process))
	path := filepath.Join(h.path, fmt.Sprintf("job-%s.lock", name))
	fl, err := flock.New(path)
	if err != nil {
		return nil, err
	}

	if err := fl.Lock(); err != nil {
		return nil, err
	}

	return fl, nil
}
