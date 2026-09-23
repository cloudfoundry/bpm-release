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
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxBoshWatcher_DetectsSpecJsonCloseWrite(t *testing.T) {
	tempBoshDir := t.TempDir()
	specEvents := make(chan struct{}, 10)

	watcher, err := startBoshDirWatcher(tempBoshDir, specEvents, log.Default())
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			t.Errorf("closing watcher: %v", err)
		}
	}()

	// Writing an unrelated file should not trigger specEvents
	otherFile := filepath.Join(tempBoshDir, "other.json")
	if err := os.WriteFile(otherFile, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing other file: %v", err)
	}

	select {
	case <-specEvents:
		t.Fatal("unexpected event received for unrelated file")
	case <-time.After(100 * time.Millisecond):
		// Expected: no event
	}

	// Writing and closing spec.json MUST trigger specEvents
	specFilePath := filepath.Join(tempBoshDir, "spec.json")
	f, err := os.OpenFile(specFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("creating spec.json: %v", err)
	}
	if _, err := f.Write([]byte(`{"deployment":"test"}`)); err != nil {
		t.Fatalf("writing spec.json: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("closing spec.json: %v", err)
	}

	select {
	case <-specEvents:
		// Success! Event caught on close
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for spec.json close-write event")
	}
}

func TestLinuxBoshWatcher_DetectsSpecJsonRename(t *testing.T) {
	tempBoshDir := t.TempDir()
	specEvents := make(chan struct{}, 10)

	watcher, err := startBoshDirWatcher(tempBoshDir, specEvents, log.Default())
	if err != nil {
		t.Fatalf("failed to start watcher: %v", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			t.Errorf("closing watcher: %v", err)
		}
	}()

	tmpFile := filepath.Join(tempBoshDir, "spec.json.tmp")
	if err := os.WriteFile(tmpFile, []byte(`{"deployment":"test"}`), 0o644); err != nil {
		t.Fatalf("writing tmp spec file: %v", err)
	}
	// Drain any event from tmp file
	select {
	case <-specEvents:
	case <-time.After(50 * time.Millisecond):
	}

	// Rename tmp file to spec.json (IN_MOVED_TO)
	specFilePath := filepath.Join(tempBoshDir, "spec.json")
	if err := os.Rename(tmpFile, specFilePath); err != nil {
		t.Fatalf("renaming to spec.json: %v", err)
	}

	select {
	case <-specEvents:
		// Success! Event caught on rename
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for spec.json rename event")
	}
}
