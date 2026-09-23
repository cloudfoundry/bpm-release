// Copyright (C) 2017-Present CloudFoundry.org Foundation, Inc. All rights reserved.
//
// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License");
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

//go:build linux

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
)

type linuxBoshWatcher struct {
	fd     int
	wd     int
	done   chan struct{}
	logger *log.Logger
}

func startBoshDirWatcher(boshDir string, events chan<- struct{}, logger *log.Logger) (io.Closer, error) {
	if err := os.MkdirAll(boshDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating bosh directory %s: %w", boshDir, err)
	}

	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC)
	if err != nil {
		return nil, fmt.Errorf("inotify_init1: %w", err)
	}

	// We watch for IN_CLOSE_WRITE (file written & closed) and IN_MOVED_TO (atomic file rename into dir)
	mask := uint32(syscall.IN_CLOSE_WRITE | syscall.IN_MOVED_TO)
	wd, err := syscall.InotifyAddWatch(fd, boshDir, mask)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("watching %s: %w", boshDir, err), syscall.Close(fd))
	}

	w := &linuxBoshWatcher{
		fd:     fd,
		wd:     wd,
		done:   make(chan struct{}),
		logger: logger,
	}

	go w.readEvents(events)

	return w, nil
}

func (w *linuxBoshWatcher) readEvents(events chan<- struct{}) {
	buf := make([]byte, 64*1024)
	for {
		select {
		case <-w.done:
			return
		default:
		}

		n, err := syscall.Read(w.fd, buf)
		if errors.Is(err, syscall.EINTR) {
			continue
		}
		if err != nil {
			select {
			case <-w.done:
				return
			default:
				// Exit so systemd restarts us with a fresh watch rather
				// than limping along on the heartbeat alone.
				logger := w.logger
				if logger == nil {
					logger = log.Default()
				}
				logger.Fatalf("reading inotify events: %v", err)
			}
		}

		// Parse inotify events looking for spec.json
		if hasSpecEvent(buf[:n]) {
			select {
			case events <- struct{}{}:
			default: // Non-blocking if channel already has an event queued
			}
		}
	}
}

func hasSpecEvent(buf []byte) bool {
	const headerLen = 16 // wd(4) + mask(4) + cookie(4) + len(4)
	offset := 0
	for offset+headerLen <= len(buf) {
		nameLen := int(binary.NativeEndian.Uint32(buf[offset+12:]))
		if offset+headerLen+nameLen > len(buf) {
			break
		}
		if nameLen > 0 {
			nameBytes := buf[offset+headerLen : offset+headerLen+nameLen]
			for i, b := range nameBytes {
				if b == 0 {
					nameBytes = nameBytes[:i]
					break
				}
			}
			name := string(nameBytes)
			if name == "spec.json" {
				return true
			}
		}
		offset += headerLen + nameLen
	}
	return false
}

func (w *linuxBoshWatcher) Close() error {
	close(w.done)
	return syscall.Close(w.fd)
}
