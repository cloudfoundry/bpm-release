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
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onsi/gomega/gexec"

	"bpm/integration2/bpmsandbox"
)

var runcExe = flag.String("runcExe", "/var/vcap/packages/bpm/bin/runc", "path to the runc executable")
var tiniExe = flag.String("tiniExe", "/var/vcap/packages/bpm/bin/tini", "path to the tini executable")
var bpmExe = flag.String("bpmExe", "", "path to bpm executable")

func TestMain(m *testing.M) {
	flag.Parse()

	// build bpm if not testing an existing binary
	if *bpmExe == "" {
		path, err := gexec.Build("bpm/cmd/bpm")
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "failed to compile bpm: %v", err)
			os.Exit(1)
		}
		*bpmExe = path
	}

	bpmsandbox.RuncPath = *runcExe
	bpmsandbox.BPMPath = *bpmExe
	bpmsandbox.TiniPath = *tiniExe

	status := m.Run()
	gexec.CleanupBuildArtifacts()
	os.Exit(status)
}

func TestRun(t *testing.T) {
	t.Parallel()
	s := bpmsandbox.New(t)
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

	stdout, err := os.ReadFile(s.Path("sys", "log", "errand", "errand.stdout.log"))
	if err != nil {
		t.Fatalf("failed to read stdout log: %v", err)
	}
	if contents, sentinel := string(stdout), "stdout"; !strings.Contains(contents, sentinel) {
		t.Errorf("stdout log file did not contain %q, contents: %q", sentinel, contents)
	}

	stderr, err := os.ReadFile(s.Path("sys", "log", "errand", "errand.stderr.log"))
	if err != nil {
		t.Fatalf("failed to read stderr log: %v", err)
	}
	if contents, sentinel := string(stderr), "stderr"; !strings.Contains(contents, sentinel) {
		t.Errorf("stderr log file did not contain %q, contents: %q", sentinel, contents)
	}

	pidfile := s.Path("sys", "run", "bpm", "errand", "errand.pid")
	if _, err := os.Stat(pidfile); !os.IsNotExist(err) {
		t.Errorf("expected %q not to exist but it did", pidfile)
	}
}

func TestRunWithEnvFlags(t *testing.T) {
	t.Parallel()
	s := bpmsandbox.New(t)
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
	s := bpmsandbox.New(t)
	defer s.Cleanup()

	s.LoadFixture("errand", "testdata/volume-flag.yml")
	extraVolumeDir := s.Path("data", "extra-volume")
	extraVolumeFile := filepath.Join(extraVolumeDir, "data.txt")

	cmd := s.BPMCmd(
		"run",
		"errand",
		"-v", fmt.Sprintf("%s:writable,allow_executions", extraVolumeDir),
		"-e", fmt.Sprintf("FILE_TO_WRITE_TO=%s", extraVolumeFile),
	)
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("failed to run bpm: %s", output)
	}
	mounts, err := parseFstab(output)
	if err != nil {
		t.Fatalf("could not parse output as fstab (%q): %q", output, err)
	}
	if len(mounts) != 1 {
		t.Fatalf("more than one mount was grepped, got: %d (%#v)", len(mounts), mounts)
	}
	firstMount := mounts[0]

	// check the path of the mount
	if have, want := firstMount.MountPoint, extraVolumeDir; have != want {
		t.Errorf("mountpoint did not contain %q, have: %q", want, have)
	}

	// check the mount has no read only option
	if mountHasOption(firstMount, "ro") {
		t.Errorf("mount contained read only option, contents: %q", firstMount.Options)
	}

	// check the mount has no noexec option
	if mountHasOption(firstMount, "noexec") {
		t.Errorf("mount contained read only option, contents: %s", firstMount.Options)
	}

	fileContents, err := os.ReadFile(extraVolumeFile)
	if err != nil {
		t.Fatalf("failed to read extra volume file: %v", err)
	}
	if contents, sentinel := string(fileContents), "success"; !strings.Contains(contents, sentinel) {
		t.Errorf("extra volume file did not contain %q, contents: %q", sentinel, contents)
	}
}

func TestRunFailure(t *testing.T) {
	t.Parallel()
	s := bpmsandbox.New(t)
	defer s.Cleanup()

	s.LoadFixture("oops", "testdata/failure.yml")

	if err := s.BPMCmd("run", "oops").Run(); err == nil {
		t.Fatal("expected command to fail but it did not")
	}
}

func TestRunUnusualExitStatus(t *testing.T) {
	t.Parallel()
	s := bpmsandbox.New(t)
	defer s.Cleanup()

	// exit status 6
	s.LoadFixture("odd", "testdata/odd-status.yml")

	cmd := s.BPMCmd("run", "odd")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected command to fail but it did not")
	}

	status := cmd.ProcessState.ExitCode()
	if status != 6 {
		t.Errorf("expected bpm to exit with status %d; got: %d", 6, status)
	}
}

type mount struct {
	MountPoint string
	Options    []string
}

// ParseFstab parses byte slices which contain the contents of files formatted
// as described by fstab(5).
func parseFstab(contents []byte) ([]mount, error) {
	var mounts []mount

	r := bytes.NewBuffer(contents)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			return nil, fmt.Errorf("invalid mount: %s", scanner.Text())
		}

		options := strings.Split(fields[3], ",")
		mounts = append(mounts, mount{
			MountPoint: fields[1],
			Options:    options,
		})
	}

	return mounts, nil
}

func mountHasOption(m mount, opt string) bool {
	for _, o := range m.Options {
		if o == opt {
			return true
		}
	}

	return false
}
