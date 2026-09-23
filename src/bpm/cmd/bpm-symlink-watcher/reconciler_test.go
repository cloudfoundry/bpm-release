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

package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func setupTestRoot(t *testing.T) (string, *Reconciler) {
	t.Helper()
	root := t.TempDir()

	usrBin := filepath.Join(root, "usr", "bin")
	if err := os.MkdirAll(usrBin, 0o755); err != nil {
		t.Fatalf("creating usr bin: %v", err)
	}

	usrBinBpm := filepath.Join(usrBin, "bpm")
	if err := os.WriteFile(usrBinBpm, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("writing stemcell bpm wrapper: %v", err)
	}

	artifacts := []Artifact{
		{
			Name:       "bpm-bin",
			LinkPath:   filepath.Join(root, "var", "vcap", "jobs", "bpm", "bin", "bpm"),
			LinkTarget: usrBinBpm,
			JobDir:     filepath.Join(root, "var", "vcap", "jobs", "bpm"),
		},
	}

	r := &Reconciler{
		Artifacts: artifacts,
	}

	return root, r
}

func TestReconcileAll_BootMissingSymlinks(t *testing.T) {
	_, r := setupTestRoot(t)

	restored, err := r.ReconcileAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(restored) != 1 || restored[0] != "bpm-bin" {
		t.Fatalf("expected 1 restored artifact, got %d: %v", len(restored), restored)
	}

	fi, err := os.Lstat(r.Artifacts[0].LinkPath)
	if err != nil {
		t.Fatalf("expected symlink %s to exist: %v", r.Artifacts[0].LinkPath, err)
	}
	if fi.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("expected %s to be a symlink", r.Artifacts[0].LinkPath)
	}
	target, err := os.Readlink(r.Artifacts[0].LinkPath)
	if err != nil {
		t.Fatalf("reading link %s: %v", r.Artifacts[0].LinkPath, err)
	}
	if target != r.Artifacts[0].LinkTarget {
		t.Fatalf("expected target %s, got %s", r.Artifacts[0].LinkTarget, target)
	}

	// Running ReconcileAll again should be a no-op since healthy symlink exists
	restoredSecondPass, err := r.ReconcileAll()
	if err != nil {
		t.Fatalf("unexpected error on second pass: %v", err)
	}
	if len(restoredSecondPass) != 0 {
		t.Fatalf("expected 0 restored artifacts on second pass, got %d: %v", len(restoredSecondPass), restoredSecondPass)
	}
}

func TestReconcileAll_DeployedVersionUntouched(t *testing.T) {
	root, r := setupTestRoot(t)

	// Simulate BOSH deployment providing bpm release:
	// /var/vcap/jobs/bpm -> /var/vcap/data/jobs/bpm/digest123
	// with a regular file at /var/vcap/data/jobs/bpm/digest123/bin/bpm
	deployedJobTarget := filepath.Join(root, "var", "vcap", "data", "jobs", "bpm", "digest123")
	if err := os.MkdirAll(filepath.Join(deployedJobTarget, "bin"), 0o755); err != nil {
		t.Fatalf("creating deployed job target: %v", err)
	}
	if err := os.WriteFile(filepath.Join(deployedJobTarget, "bin", "bpm"), []byte("deployed-bpm"), 0o755); err != nil {
		t.Fatalf("writing deployed bpm: %v", err)
	}

	jobsDir := filepath.Join(root, "var", "vcap", "jobs")
	if err := os.MkdirAll(jobsDir, 0o755); err != nil {
		t.Fatalf("mkdir jobs dir: %v", err)
	}

	if err := os.Symlink(deployedJobTarget, filepath.Join(jobsDir, "bpm")); err != nil {
		t.Fatalf("symlinking deployed job: %v", err)
	}

	restored, err := r.ReconcileAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(restored) != 0 {
		t.Fatalf("expected 0 restored artifacts, got %d: %v", len(restored), restored)
	}

	// Verify link path reaches deployed regular file
	fi, err := os.Lstat(r.Artifacts[0].LinkPath)
	if err != nil {
		t.Fatalf("expected linkPath to resolve: %v", err)
	}
	if fi.Mode()&fs.ModeSymlink != 0 {
		t.Fatalf("expected leaf to be regular file reached via symlink, got symlink")
	}
}

func TestReconcileAll_DanglingSymlinkRestored(t *testing.T) {
	root, r := setupTestRoot(t)

	// Simulate leaf symlink pointing to nonexistent target
	danglingTarget := filepath.Join(root, "nonexistent", "bpm")
	if err := os.MkdirAll(filepath.Dir(r.Artifacts[0].LinkPath), 0o755); err != nil {
		t.Fatalf("mkdir jobs dir: %v", err)
	}
	if err := os.Symlink(danglingTarget, r.Artifacts[0].LinkPath); err != nil {
		t.Fatalf("symlinking dangling bpm: %v", err)
	}

	if _, err := os.Stat(r.Artifacts[0].LinkPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected Stat to fail with ErrNotExist on dangling link, got: %v", err)
	}

	restored, err := r.ReconcileAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(restored) != 1 || restored[0] != "bpm-bin" {
		t.Fatalf("expected artifact to be restored, got %d: %v", len(restored), restored)
	}

	target, err := os.Readlink(r.Artifacts[0].LinkPath)
	if err != nil || target != r.Artifacts[0].LinkTarget {
		t.Fatalf("expected link %s to point to %s, got %s (err: %v)", r.Artifacts[0].LinkPath, r.Artifacts[0].LinkTarget, target, err)
	}
}

func TestReconcileAll_MissingFallbackNotRewritten(t *testing.T) {
	_, r := setupTestRoot(t)

	// Stemcell has no /usr/bin/bpm, so the fallback link dangles
	if err := os.Remove(r.Artifacts[0].LinkTarget); err != nil {
		t.Fatalf("removing stemcell bpm wrapper: %v", err)
	}

	if _, err := r.ReconcileAll(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// A second pass must not rewrite a link that already points at the fallback
	restored, err := r.ReconcileAll()
	if err != nil {
		t.Fatalf("unexpected error on second pass: %v", err)
	}
	if len(restored) != 0 {
		t.Fatalf("expected 0 restored artifacts on second pass, got %d: %v", len(restored), restored)
	}

	target, err := os.Readlink(r.Artifacts[0].LinkPath)
	if err != nil || target != r.Artifacts[0].LinkTarget {
		t.Fatalf("expected link %s to point to %s, got %s (err: %v)", r.Artifacts[0].LinkPath, r.Artifacts[0].LinkTarget, target, err)
	}
}

func TestReconcileAll_DanglingParentRestored(t *testing.T) {
	root, r := setupTestRoot(t)

	// Simulate /var/vcap/jobs/bpm itself being a dangling symlink
	// (e.g. after agent crash between Disable() and Uninstall())
	danglingParentTarget := filepath.Join(root, "var", "vcap", "data", "jobs", "bpm", "removed-digest")
	jobsBpmDir := filepath.Join(root, "var", "vcap", "jobs", "bpm")

	if err := os.MkdirAll(filepath.Dir(jobsBpmDir), 0o755); err != nil {
		t.Fatalf("mkdir jobs dir: %v", err)
	}
	if err := os.Symlink(danglingParentTarget, jobsBpmDir); err != nil {
		t.Fatalf("symlinking dangling jobs/bpm: %v", err)
	}

	// Confirm parent is a dangling symlink
	if _, err := os.Lstat(jobsBpmDir); err != nil {
		t.Fatalf("expected Lstat to succeed on dangling parent: %v", err)
	}
	if _, err := os.Stat(jobsBpmDir); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("expected Stat to fail on dangling parent, got: %v", err)
	}

	restored, err := r.ReconcileAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(restored) != 1 || restored[0] != "bpm-bin" {
		t.Fatalf("expected 1 restored artifact, got %d: %v", len(restored), restored)
	}

	// Verify /var/vcap/jobs/bpm is now a real directory
	fi, err := os.Lstat(jobsBpmDir)
	if err != nil || !fi.IsDir() {
		t.Fatalf("expected %s to be directory, got fi=%v, err=%v", jobsBpmDir, fi, err)
	}

	// Verify link was restored and resolves
	target, err := os.Readlink(r.Artifacts[0].LinkPath)
	if err != nil || target != r.Artifacts[0].LinkTarget {
		t.Fatalf("expected link %s to point to %s, got %s (err: %v)", r.Artifacts[0].LinkPath, r.Artifacts[0].LinkTarget, target, err)
	}
	if _, err := os.Stat(r.Artifacts[0].LinkPath); err != nil {
		t.Fatalf("expected link %s to resolve successfully: %v", r.Artifacts[0].LinkPath, err)
	}
}

func TestReconcileAll_NonSymlinkUntouched(t *testing.T) {
	_, r := setupTestRoot(t)

	// Create a regular file at linkPath
	if err := os.MkdirAll(filepath.Dir(r.Artifacts[0].LinkPath), 0o755); err != nil {
		t.Fatalf("creating dir at link path: %v", err)
	}
	if err := os.WriteFile(r.Artifacts[0].LinkPath, []byte("regular-file"), 0o755); err != nil {
		t.Fatalf("writing regular file: %v", err)
	}

	restored, err := r.ReconcileAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(restored) != 0 {
		t.Fatalf("expected 0 restored artifacts, got: %v", restored)
	}

	fi, err := os.Lstat(r.Artifacts[0].LinkPath)
	if err != nil || fi.Mode()&fs.ModeSymlink != 0 {
		t.Fatalf("expected %s to remain a regular file, got fi=%v, err=%v", r.Artifacts[0].LinkPath, fi, err)
	}
}

func TestReconcileAll_UnresolvableSymlinkRestored(t *testing.T) {
	root, r := setupTestRoot(t)

	if err := os.MkdirAll(filepath.Dir(r.Artifacts[0].LinkPath), 0o755); err != nil {
		t.Fatalf("mkdir jobs dir: %v", err)
	}
	notADir := filepath.Join(root, "not-a-dir")
	if err := os.WriteFile(notADir, nil, 0o644); err != nil {
		t.Fatalf("writing not-a-dir: %v", err)
	}

	for name, linkTarget := range map[string]string{
		"loop":    r.Artifacts[0].LinkPath,       // Stat fails with ELOOP
		"notadir": filepath.Join(notADir, "bpm"), // Stat fails with ENOTDIR
	} {
		t.Run(name, func(t *testing.T) {
			if err := os.Remove(r.Artifacts[0].LinkPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("removing link: %v", err)
			}
			if err := os.Symlink(linkTarget, r.Artifacts[0].LinkPath); err != nil {
				t.Fatalf("symlinking broken bpm: %v", err)
			}

			restored, err := r.ReconcileAll()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(restored) != 1 || restored[0] != "bpm-bin" {
				t.Fatalf("expected artifact to be restored, got %d: %v", len(restored), restored)
			}

			target, err := os.Readlink(r.Artifacts[0].LinkPath)
			if err != nil || target != r.Artifacts[0].LinkTarget {
				t.Fatalf("expected link %s to point to %s, got %s (err: %v)", r.Artifacts[0].LinkPath, r.Artifacts[0].LinkTarget, target, err)
			}
		})
	}
}

func TestSymlinkIfAbsent_ExistingPathUntouched(t *testing.T) {
	_, r := setupTestRoot(t)

	// Simulate the agent installing bin/bpm after Reconcile saw it missing
	if err := os.MkdirAll(filepath.Dir(r.Artifacts[0].LinkPath), 0o755); err != nil {
		t.Fatalf("mkdir jobs dir: %v", err)
	}
	if err := os.WriteFile(r.Artifacts[0].LinkPath, []byte("deployed-bpm"), 0o755); err != nil {
		t.Fatalf("writing deployed bpm: %v", err)
	}

	created, err := symlinkIfAbsent(r.Artifacts[0].LinkTarget, r.Artifacts[0].LinkPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Fatalf("expected existing path to be left alone")
	}

	contents, err := os.ReadFile(r.Artifacts[0].LinkPath)
	if err != nil || string(contents) != "deployed-bpm" {
		t.Fatalf("expected deployed bpm to be untouched, got %q (err: %v)", contents, err)
	}
}
