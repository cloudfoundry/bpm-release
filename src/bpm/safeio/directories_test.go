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

package safeio_test

import (
	"os"
	"path/filepath"
	"testing"

	"bpm/safeio"
)

func TestVolumePreparationDoesNotFollowUserSymlinks(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0777); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "outside")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(target, "sentinel")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0644); err != nil {
		t.Fatal(err)
	}
	leaf := filepath.Join(root, "volume")
	if err := os.Symlink(target, leaf); err != nil {
		t.Fatal(err)
	}
	for _, prepare := range []func(string, int, int) error{safeio.MkdirChown, safeio.PrepareVolume} {
		if err := prepare(leaf, os.Getuid(), os.Getgid()); err == nil {
			t.Fatal("leaf symlink accepted")
		}
		if err := prepare(filepath.Join(leaf, "nested"), os.Getuid(), os.Getgid()); err == nil {
			t.Fatal("ancestor symlink accepted")
		}
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("outside directory mode changed: %v", info.Mode())
	}
	data, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "unchanged" {
		t.Fatal("outside content changed")
	}
	if _, err := os.Stat(filepath.Join(target, "nested")); !os.IsNotExist(err) {
		t.Fatal("created directory through symlink")
	}
}

func TestVolumePreparationRejectsLinkedFileAndCreatesDirectories(t *testing.T) {
	root := t.TempDir()
	sentinel := filepath.Join(root, "outside")
	linked := filepath.Join(root, "linked")
	if err := os.WriteFile(sentinel, []byte("unchanged"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(sentinel, linked); err != nil {
		t.Fatal(err)
	}
	if err := safeio.PrepareVolume(linked, os.Getuid(), os.Getgid()); err == nil {
		t.Fatal("multiply linked file accepted")
	}
	created := filepath.Join(root, "new", "nested")
	if err := safeio.MkdirChown(created, os.Getuid(), os.Getgid()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(created)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatalf("incorrect directory mode: %v", info.Mode())
	}
	if err := safeio.PrepareVolume(filepath.Join(root, "newvolume"), os.Getuid(), os.Getgid()); err != nil {
		t.Fatal(err)
	}
}
