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

package flock

import (
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

// Flock represents a handle on a file which can then be locked and unlocked to
// provide cross-process synchronization and locking.
type Flock struct {
	f *os.File

	locked   bool
	lockedMu sync.Mutex
}

// New creates a new handle on a file. If the file does not exist then it is
// created. The parent directory must already exist before this is called.
func New(path string) (*Flock, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	return &Flock{
		f: f,
	}, nil
}

// Lock exclusively locks the file. Subsequent calls will block until the
// original lock is released. It is safe to call this concurrently in both the
// current process and other processes.
func (f *Flock) Lock() error {
	f.lockedMu.Lock()
	defer f.lockedMu.Unlock()

	err := unix.Flock(int(f.f.Fd()), unix.LOCK_EX)
	if err != nil {
		return err
	}

	f.locked = true
	return nil
}

// Unlock unlocks the file so that another waiting task can acquire the lock.
// This function will panic unlock is called on a lock which is already
// unlocked. It is not possible to unlock a lock from a different handle than
// locked it.
func (f *Flock) Unlock() error {
	f.lockedMu.Lock()
	defer f.lockedMu.Unlock()
	if !f.locked {
		// This is such a mistake that returning an error doesn't quite cut it.
		panic("flock: unlock of unlocked lock")
	}

	err := unix.Flock(int(f.f.Fd()), unix.LOCK_UN)
	if err != nil {
		return err
	}

	f.locked = false
	return nil
}
