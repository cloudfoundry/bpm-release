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
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

type Artifact struct {
	Name       string
	LinkPath   string
	LinkTarget string
	JobDir     string
}

var DefaultArtifacts = []Artifact{
	{
		Name:       "bpm-bin",
		LinkPath:   "/var/vcap/jobs/bpm/bin/bpm",
		LinkTarget: "/usr/bin/bpm",
		JobDir:     "/var/vcap/jobs/bpm",
	},
}

type Reconciler struct {
	Artifacts []Artifact
	Logger    *log.Logger
}

func NewReconciler(artifacts []Artifact, logger *log.Logger) *Reconciler {
	if logger == nil {
		logger = log.Default()
	}
	return &Reconciler{
		Artifacts: artifacts,
		Logger:    logger,
	}
}

func (r *Reconciler) ReconcileAll() ([]string, error) {
	var restored []string
	for _, a := range r.Artifacts {
		didRestore, err := r.Reconcile(a)
		if err != nil {
			return restored, fmt.Errorf("reconciling %s: %w", a.Name, err)
		}
		if didRestore {
			restored = append(restored, a.Name)
		}
	}
	return restored, nil
}

func (r *Reconciler) Reconcile(a Artifact) (bool, error) {
	fi, err := os.Lstat(a.LinkPath)
	if errors.Is(err, fs.ErrNotExist) {
		if r.Logger != nil {
			r.Logger.Printf("artifact %s (%s) does not exist; restoring fallback %s", a.Name, a.LinkPath, a.LinkTarget)
		}
		if err := removeDanglingSymlink(a.JobDir); err != nil {
			return false, err
		}
		created, err := symlinkIfAbsent(a.LinkTarget, a.LinkPath)
		if err != nil {
			return false, err
		}
		if !created {
			if r.Logger != nil {
				r.Logger.Printf("artifact %s (%s) was installed concurrently; leaving it", a.Name, a.LinkPath)
			}
			return false, nil
		}
	} else if err != nil {
		return false, err
	} else {
		// If it's a regular file or directory, leave it alone (not our symlink to touch)
		if fi.Mode()&fs.ModeSymlink == 0 {
			return false, nil
		}

		// Symlink exists: check if target is valid and reachable
		if _, statErr := os.Stat(a.LinkPath); statErr == nil {
			// Target exists; deployed or fallback symlink is healthy
			return false, nil
		} else if !unresolvable(statErr) {
			return false, statErr
		}

		// Already the fallback, but the fallback itself is missing (no
		// stemcell bpm). Rewriting the link would not help, so leave it.
		if target, err := os.Readlink(a.LinkPath); err == nil && target == a.LinkTarget {
			return false, nil
		}

		// Symlink exists but can never resolve (dangling link or loop)
		if r.Logger != nil {
			r.Logger.Printf("artifact %s (%s) has broken symlink; restoring fallback %s", a.Name, a.LinkPath, a.LinkTarget)
		}
		if err := atomicSymlink(a.LinkTarget, a.LinkPath); err != nil {
			return false, err
		}
	}

	if r.Logger != nil {
		r.Logger.Printf("successfully restored %s -> %s", a.LinkPath, a.LinkTarget)
	}
	return true, nil
}

// unresolvable reports whether a Stat error means the link's target cannot be
// reached, as opposed to a transient failure where the target may exist.
func unresolvable(err error) bool {
	return errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ELOOP) || errors.Is(err, syscall.ENOTDIR)
}

func removeDanglingSymlink(path string) error {
	if path == "" {
		return nil
	}
	fi, err := os.Lstat(path)
	if err != nil || fi.Mode()&fs.ModeSymlink == 0 {
		return nil
	}
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return os.Remove(path)
}

// symlinkIfAbsent creates linkPath without replacing anything, so a file or
// link the agent installs after the Lstat in Reconcile is left alone. It
// reports false if linkPath already exists.
func symlinkIfAbsent(target, linkPath string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		return false, err
	}
	err := os.Symlink(target, linkPath)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	return err == nil, err
}

func atomicSymlink(target, linkPath string) error {
	dir := filepath.Dir(linkPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	tmp := filepath.Join(dir, ".tmp."+filepath.Base(linkPath))
	if err := os.Remove(tmp); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if err := os.Symlink(target, tmp); err != nil {
		return err
	}

	if err := os.Rename(tmp, linkPath); err != nil {
		return errors.Join(err, os.Remove(tmp))
	}

	return nil
}
