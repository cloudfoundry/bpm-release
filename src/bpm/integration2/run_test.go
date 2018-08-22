// Copyright (C) 2018-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package integration2_test

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/onsi/gomega/gexec"

	"bpm/mount"
)

var runcExe = flag.String("runcExe", "/var/vcap/packages/bpm/bin/runc", "path to the runc executable")
var bpmExe = flag.String("bpmExe", "", "path to bpm executable")

func TestMain(m *testing.M) {
	flag.Parse()

	// build bpm if not testing an existing binary
	if *bpmExe == "" {
		path, err := gexec.Build("bpm/cmd/bpm")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to compile bpm: %v", err)
			os.Exit(1)
		}
		*bpmExe = path
	}

	os.Exit(m.Run())
}

func TestRun(t *testing.T) {
	t.Parallel()
	s := NewSandbox(t)
	defer s.Cleanup()

	s.LoadFixture("errand", "testdata/errand.yml")

	cmd := s.BPMCmd("run", "errand")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run bpm: %s", output)
	}

	if contents, sentinel := string(output), "stdout"; !strings.Contains(contents, sentinel) {
		t.Errorf("stdout/stderr did not contain %q, contents: %q", sentinel, contents)
	}
	if contents, sentinel := string(output), "stderr"; !strings.Contains(contents, sentinel) {
		t.Errorf("stdout/stderr did not contain %q, contents: %q", sentinel, contents)
	}
	stdout, err := ioutil.ReadFile(s.Path("sys", "log", "errand", "errand.stdout.log"))
	if err != nil {
		t.Fatalf("failed to read stdout log: %v", err)
	}
	if contents, sentinel := string(stdout), "stdout"; !strings.Contains(contents, sentinel) {
		t.Errorf("stdout log file did not contain %q, contents: %q", sentinel, contents)
	}
	stderr, err := ioutil.ReadFile(s.Path("sys", "log", "errand", "errand.stderr.log"))
	if err != nil {
		t.Fatalf("failed to read stderr log: %v", err)
	}
	if contents, sentinel := string(stderr), "stderr"; !strings.Contains(contents, sentinel) {
		t.Errorf("stderr log file did not contain %q, contents: %q", sentinel, contents)
	}
}

func TestRunWithEnvFlags(t *testing.T) {
	t.Parallel()
	s := NewSandbox(t)
	defer s.Cleanup()

	s.LoadFixture("errand", "testdata/env-flag.yml")
	sentinel := "sentinel"

	cmd := s.BPMCmd(
		"run",
		"errand",
		"-e", fmt.Sprintf("ENVKEY=%s", sentinel),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run bpm: %s", output)
	}
	if contents, sentinel := string(output), sentinel; !strings.Contains(contents, sentinel) {
		t.Errorf("output did not contain %q; contents: %q", sentinel, contents)
	}
}

func TestRunWithVolumeFlags(t *testing.T) {
	t.Parallel()
	s := NewSandbox(t)
	defer s.Cleanup()

	s.LoadFixture("errand", "testdata/volume-flag.yml")
	extraVolumeDir := filepath.Join(s.root, "data", "extra-volume")
	extraVolumeFile := filepath.Join(extraVolumeDir, "data.txt")

	cmd := s.BPMCmd(
		"run",
		"errand",
		"-v", fmt.Sprintf("%s:writable,allow_executions", extraVolumeDir),
		"-e", fmt.Sprintf("FILE_TO_WRITE_TO=%s", extraVolumeFile),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to run bpm: %s", output)
	}
	mounts, err := mount.ParseFstab(output)
	if err != nil {
		t.Fatalf("could not parse output as fstab (%q): %q", output, err)
	}
	if len(mounts) != 1 {
		t.Fatalf("more than one mount was grepped, got: %d", len(mounts))
	}
	mnt := mounts[0]

	// check the path of the mount
	if have, want := mnt.MountPoint, extraVolumeDir; have != want {
		t.Errorf("mountpoint did not contain %q, have: %q", want, have)
	}

	// check the mount has no read only option
	if mountHasOption(mnt, "ro") {
		t.Errorf("mount contained read only option, contents: %q", mnt.Options)
	}

	// check the mount has no noexec option
	if mountHasOption(mnt, "noexec") {
		t.Errorf("mount contained read only option, contents: %s", mnt.Options)
	}

	fileContents, err := ioutil.ReadFile(extraVolumeFile)
	if err != nil {
		t.Fatalf("failed to read extra volume file: %v", err)
	}
	if contents, sentinel := string(fileContents), "success"; !strings.Contains(contents, sentinel) {
		t.Errorf("extra volume file did not contain %q, contents: %q", sentinel, contents)
	}
}

func mountHasOption(m mount.Mnt, opt string) bool {
	for _, o := range m.Options {
		if o == opt {
			return true
		}
	}

	return false
}

func TestRunFailure(t *testing.T) {
	t.Parallel()
	s := NewSandbox(t)
	defer s.Cleanup()

	s.LoadFixture("oops", "testdata/failure.yml")

	if err := s.BPMCmd("run", "oops").Run(); err == nil {
		t.Fatal("expected command to fail but it did not")
	}
}

func TestRunUnusualExitStatus(t *testing.T) {
	t.Parallel()
	s := NewSandbox(t)
	defer s.Cleanup()

	// exit status 6
	s.LoadFixture("odd", "testdata/odd-status.yml")

	cmd := s.BPMCmd("run", "odd")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected command to fail but it did not")
	}

	status := cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
	if status != 6 {
		t.Errorf("expected bpm to exit with status %d; got: %d", 6, status)
	}
}

type Sandbox struct {
	t *testing.T

	bpmExe  string
	runcExe string

	root string
}

func NewSandbox(t *testing.T) *Sandbox {
	root, err := ioutil.TempDir("", "bpm_sandbox")
	if err != nil {
		t.Fatalf("could not create sandbox root directory: %v", err)
	}

	paths := []string{
		filepath.Join(root, "packages", "bpm", "bin"),
		filepath.Join(root, "data", "packages"),
		filepath.Join(root, "sys", "log"),
	}

	for _, path := range paths {
		if err := os.MkdirAll(path, 0777); err != nil {
			t.Fatalf("could not create sandbox directory structure: %v", err)
		}
	}

	runcSandboxPath := filepath.Join(root, "packages", "bpm", "bin", "runc")
	if err := os.Symlink(*runcExe, runcSandboxPath); err != nil {
		t.Fatalf("could not link runc executable into sandbox: %v", err)
	}

	return &Sandbox{
		t:       t,
		bpmExe:  *bpmExe,
		runcExe: *runcExe,
		root:    root,
	}
}

func (s *Sandbox) Path(fragments ...string) string {
	return filepath.Join(append([]string{s.root}, fragments...)...)
}

func (s *Sandbox) BPMCmd(args ...string) *exec.Cmd {
	cmd := exec.Command(s.bpmExe, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("BPM_BOSH_ROOT=%s", s.root))
	return cmd
}

func (s *Sandbox) LoadFixture(job, path string) {
	configPath := filepath.Join(s.root, "jobs", job, "config", "bpm.yml")

	if err := os.MkdirAll(filepath.Dir(configPath), 0777); err != nil {
		s.t.Fatalf("failed to create fixture destination directory: %v", err)
	}

	src, err := os.Open(path)
	if err != nil {
		s.t.Fatalf("failed to open fixture source: %v", err)
	}
	defer src.Close()

	dst, err := os.Create(configPath)
	if err != nil {
		s.t.Fatalf("failed to open fixture destination: %v", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		s.t.Fatalf("failed to copy fixture to destination: %v", err)
	}
}

func (s *Sandbox) Cleanup() {
	_ = os.RemoveAll(s.root)
}
