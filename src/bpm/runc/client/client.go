// Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
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
	"io"
	"os"
	"os/exec"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type Signal int

const (
	Term Signal = iota
	Quit
)

func (s Signal) String() string {
	switch s {
	case Term:
		return "TERM"
	case Quit:
		return "QUIT"
	default:
		return "unknown"
	}
}

//go:generate counterfeiter . RuncClient

type RuncClient interface {
	CreateBundle(bundlePath string, jobSpec specs.Spec, user specs.User) error
	RunContainer(pidFilePath, bundlePath, containerID string, stdout, stderr io.Writer) error
	ContainerState(containerID string) (specs.State, error)
	ListContainers() ([]ContainerState, error)
	SignalContainer(containerID string, signal Signal) error
	DeleteContainer(containerID string) error
	DestroyBundle(bundlePath string) error
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

type runcClient struct {
	runcPath string
	runcRoot string
}

func NewRuncClient(runcPath, runcRoot string) RuncClient {
	return &runcClient{
		runcPath: runcPath,
		runcRoot: runcRoot,
	}
}

func (*runcClient) CreateBundle(bundlePath string, jobSpec specs.Spec, user specs.User) error {
	err := os.MkdirAll(bundlePath, 0700)
	if err != nil {
		return err
	}

	rootfsPath := filepath.Join(bundlePath, "rootfs")
	err = os.MkdirAll(rootfsPath, 0700)
	if err != nil {
		return err
	}

	err = os.Chown(rootfsPath, int(user.UID), int(user.GID))
	if err != nil {
		panic(err)
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

func (c *runcClient) RunContainer(pidFilePath, bundlePath, containerID string, stdout, stderr io.Writer) error {
	runcCmd := exec.Command(
		c.runcPath,
		"--root", c.runcRoot,
		"run",
		"--bundle", bundlePath,
		"--pid-file", pidFilePath,
		"--detach",
		containerID,
	)

	runcCmd.Stdout = stdout
	runcCmd.Stderr = stderr

	return runcCmd.Run()
}

func (c *runcClient) ContainerState(containerID string) (specs.State, error) {
	runcCmd := exec.Command(
		c.runcPath,
		"--root", c.runcRoot,
		"state",
		containerID,
	)

	var state specs.State
	data, err := runcCmd.CombinedOutput()
	if err != nil {
		return specs.State{}, err
	}

	err = json.Unmarshal(data, &state)
	if err != nil {
		return specs.State{}, err
	}

	return state, nil
}

func (c *runcClient) ListContainers() ([]ContainerState, error) {
	runcCmd := exec.Command(
		c.runcPath,
		"--root", c.runcRoot,
		"list",
		"--format", "json",
	)

	data, err := runcCmd.CombinedOutput()
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

func (c *runcClient) SignalContainer(containerID string, signal Signal) error {
	runcCmd := exec.Command(
		c.runcPath,
		"--root", c.runcRoot,
		"kill",
		containerID,
		signal.String(),
	)

	return runcCmd.Run()
}

func (c *runcClient) DeleteContainer(containerID string) error {
	runcCmd := exec.Command(
		c.runcPath,
		"--root", c.runcRoot,
		"delete",
		"-f",
		containerID,
	)

	return runcCmd.Run()
}

func (*runcClient) DestroyBundle(bundlePath string) error {
	return os.RemoveAll(bundlePath)
}
