// Copyright (C) 2017-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type Signal int

const (
	Term Signal = iota
	Quit
	Int
)

func (s Signal) String() string {
	switch s {
	case Term:
		return "TERM"
	case Quit:
		return "QUIT"
	case Int:
		return "INT"
	default:
		return "unknown"
	}
}

// https://github.com/opencontainers/runc/blob/master/list.go#L24-L45
type ContainerState struct {
	// ID is the container ID
	ID string `json:"id"`
	// InitProcessPid is the init process id in the parent namespace
	InitProcessPid int `json:"pid"`
	// Status is the current status of the container, running, paused, ...
	Status string `json:"status"`
}

type RuncClient struct {
	runcPath string
	runcRoot string

	inSystemd bool
}

func NewRuncClient(runcPath, runcRoot string, inSystemd bool) *RuncClient {
	return &RuncClient{
		runcPath:  runcPath,
		runcRoot:  runcRoot,
		inSystemd: inSystemd,
	}
}

func (*RuncClient) CreateBundle(
	bundlePath string,
	jobSpec specs.Spec,
	user specs.User,
) error {
	err := os.MkdirAll(bundlePath, 0700)
	if err != nil {
		return err
	}

	rootfsPath := filepath.Join(bundlePath, "rootfs")
	err = os.MkdirAll(rootfsPath, 0755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(bundlePath, "config.json"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		// This is super hard to test as we are root.
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	return enc.Encode(&jobSpec)
}

func (c *RuncClient) RunContainer(pidFilePath, bundlePath, containerID string, detach bool, stdout, stderr io.Writer) (int, error) {
	args := []string{
		"--bundle", bundlePath,
	}
	if detach {
		args = append(args, "--pid-file", pidFilePath)
		args = append(args, "--detach")
	}
	args = append(args, containerID)

	runcCmd := c.buildCmd("run", args...)
	runcCmd.Stdout = stdout
	runcCmd.Stderr = stderr

	if err := runcCmd.Run(); err != nil {
		if status, ok := runcCmd.ProcessState.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus(), err
		}

		// If we can't get the exit status for some reason then make
		// sure to at least return a generic failure.
		return 1, err
	}

	return 0, nil
}

// Exec assumes you are launching an interactive shell.
// We should improve the interface to mirror `runc exec` more generally.
func (c *RuncClient) Exec(containerID, command string, stdin io.Reader, stdout, stderr io.Writer) error {
	runcCmd := c.buildCmd(
		"exec",
		"--tty",
		"--env", fmt.Sprintf("TERM=%s", os.Getenv("TERM")),
		containerID,
		command,
	)

	runcCmd.Stdin = stdin
	runcCmd.Stdout = stdout
	runcCmd.Stderr = stderr

	return runcCmd.Run()
}

// ContainerState returns the following:
// - state, nil if the job is running,and no errors were encountered.
// - nil,nil if the container state is not running and no other errors were encountered
// - nil,error if there is any other error getting the container state
//   (e.g. the container is running but in an unreachable state)
func (c *RuncClient) ContainerState(containerID string) (*specs.State, error) {
	runcCmd := c.buildCmd(
		"--log-format",
		"json",
		"state",
		containerID,
	)

	var state specs.State
	data, err := runcCmd.CombinedOutput()
	if err != nil {
		return nil, decodeContainerStateErr(data, err)
	}

	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

func decodeContainerStateErr(b []byte, err error) error {
	var jsonErr struct {
		Msg string
	}
	e := json.Unmarshal(b, &jsonErr)
	if e != nil {
		return err
	}
	r := regexp.MustCompile(`\s*container "[^"]*" does not exist\s*`)
	if r.MatchString(jsonErr.Msg) {
		return nil
	}
	return err
}

func (c *RuncClient) ListContainers() ([]ContainerState, error) {
	runcCmd := c.buildCmd(
		"list",
		"--format", "json",
	)

	data, err := runcCmd.Output()
	if err != nil {
		return []ContainerState{}, err
	}

	var containerStates []ContainerState
	err = json.Unmarshal(data, &containerStates)
	if err != nil {
		return []ContainerState{}, err
	}

	return containerStates, nil
}

func (c *RuncClient) SignalContainer(containerID string, signal Signal) error {
	runcCmd := c.buildCmd(
		"kill",
		containerID,
		signal.String(),
	)

	return runcCmd.Run()
}

func (c *RuncClient) DeleteContainer(containerID string) error {
	runcCmd := c.buildCmd(
		"delete",
		"--force",
		containerID,
	)

	return runcCmd.Run()
}

func (*RuncClient) DestroyBundle(bundlePath string) error {
	return os.RemoveAll(bundlePath)
}

func (c *RuncClient) buildCmd(command string, extra ...string) *exec.Cmd {
	args := []string{"--root", c.runcRoot}
	if c.inSystemd {
		args = append(args, "--systemd-cgroup")
	}
	args = append(args, command)
	args = append(args, extra...)
	return exec.Command(c.runcPath, args...)
}
