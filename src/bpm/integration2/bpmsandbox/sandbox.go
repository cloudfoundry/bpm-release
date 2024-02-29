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

package bpmsandbox

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var (
	RuncPath string
	BPMPath  string
	TiniPath string
)

type Sandbox struct {
	t *testing.T

	root string
}

func New(t *testing.T) *Sandbox {
	t.Helper()

	root, err := os.MkdirTemp("", "bpm_sandbox")
	if err != nil {
		t.Fatalf("could not create sandbox root directory: %v", err)
	}

	t.Logf("created sandbox in %s", root)

	paths := []string{
		filepath.Join(root, "packages", "bpm", "bin"),
		filepath.Join(root, "data", "packages"),
		filepath.Join(root, "sys", "log"),
		filepath.Join(root, "sys", "run"),
	}

	for _, path := range paths {
		if err := os.MkdirAll(path, 0777); err != nil {
			t.Fatalf("could not create sandbox directory structure: %v", err)
		}
	}

	runcSandboxPath := filepath.Join(root, "packages", "bpm", "bin", "runc")
	if err := os.Symlink(RuncPath, runcSandboxPath); err != nil {
		t.Fatalf("could not link runc executable into sandbox: %v", err)
	}

	tiniSandboxPath := filepath.Join(root, "packages", "bpm", "bin", "tini")
	if err := copyFile(tiniSandboxPath, TiniPath); err != nil {
		t.Fatalf("could not copy tini executable into sandbox: %v", err)
	}
	if err := os.Chown(tiniSandboxPath, 2000, 3000); err != nil {
		t.Fatalf("could not chown tini executable: %v", err)
	}
	if err := os.Chmod(tiniSandboxPath, 0700); err != nil {
		t.Fatalf("could not chown tini executable: %v", err)
	}

	return &Sandbox{
		t:    t,
		root: root,
	}
}

func (s *Sandbox) Path(fragments ...string) string {
	return filepath.Join(append([]string{s.root}, fragments...)...)
}

func (s *Sandbox) BPMCmd(args ...string) *exec.Cmd {
	cmd := exec.Command(BPMPath, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("BPM_BOSH_ROOT=%s", s.root))
	return cmd
}

func (s *Sandbox) LoadFixture(job, path string) {
	configPath := filepath.Join(s.root, "jobs", job, "config", "bpm.yml")

	if err := os.MkdirAll(filepath.Dir(configPath), 0777); err != nil {
		s.t.Fatalf("failed to create fixture destination directory: %v", err)
	}

	if err := copyFile(configPath, path); err != nil {
		s.t.Fatalf("failed to copy file: %v", err)
	}
}

func (s *Sandbox) Cleanup() {
	s.t.Helper()
	s.t.Logf("deleting sandbox %s", s.root)
	_ = os.RemoveAll(s.root)
}

func copyFile(dst, src string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(d, s); err != nil {
		d.Close()
		return err
	}
	return d.Close()
}
