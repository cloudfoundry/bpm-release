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

package lifecycle

import (
	"bpm/bpm"
	"bpm/models"
	"bpm/runc/client"
	"bpm/usertools"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

const (
	ContainerSigQuitGracePeriod = 5 * time.Second
	ContainerStatePollInterval  = 1 * time.Second
)

var TimeoutError = errors.New("failed to stop job within timeout")

//go:generate counterfeiter . UserFinder

type UserFinder interface {
	Lookup(username string) (specs.User, error)
}

//go:generate counterfeiter . RuncAdapter

type RuncAdapter interface {
	CreateJobPrerequisites(systemRoot, jobName, procName string, user specs.User) (string, *os.File, *os.File, error)
	BuildSpec(systemRoot, jobName, procName string, cfg *bpm.Config, user specs.User) (specs.Spec, error)
}

//go:generate counterfeiter . RuncClient

type RuncClient interface {
	CreateBundle(bundlePath string, jobSpec specs.Spec, user specs.User) error
	RunContainer(pidFilePath, bundlePath, containerID string, stdout, stderr io.Writer) error
	Exec(containerID, command string, stdin io.Reader, stdout, stderr io.Writer) error
	ContainerState(containerID string) (*specs.State, error)
	ListContainers() ([]client.ContainerState, error)
	SignalContainer(containerID string, signal client.Signal) error
	DeleteContainer(containerID string) error
	DestroyBundle(bundlePath string) error
}

type RuncLifecycle struct {
	clock       clock.Clock
	runcClient  RuncClient
	runcAdapter RuncAdapter
	systemRoot  string
	userFinder  UserFinder
}

func NewRuncLifecycle(
	runcClient RuncClient,
	runcAdapter RuncAdapter,
	userFinder UserFinder,
	clock clock.Clock,
	systemRoot string,
) *RuncLifecycle {
	return &RuncLifecycle{
		clock:       clock,
		runcClient:  runcClient,
		runcAdapter: runcAdapter,
		systemRoot:  systemRoot,
		userFinder:  userFinder,
	}
}

func (j *RuncLifecycle) StartJob(jobName, procName string, cfg *bpm.Config) error {
	user, err := j.userFinder.Lookup(usertools.VcapUser)
	if err != nil {
		return err
	}

	pidDir, stdout, stderr, err := j.runcAdapter.CreateJobPrerequisites(j.systemRoot, jobName, procName, user)
	if err != nil {
		return fmt.Errorf("failed to create system files: %s", err.Error())
	}
	defer stdout.Close()
	defer stderr.Close()

	spec, err := j.runcAdapter.BuildSpec(j.systemRoot, jobName, procName, cfg, user)
	if err != nil {
		return err
	}

	bundlePath := j.bundlePath(jobName, procName)
	err = j.runcClient.CreateBundle(bundlePath, spec, user)
	if err != nil {
		return fmt.Errorf("bundle build failure: %s", err.Error())
	}

	pidFilePath := filepath.Join(pidDir, fmt.Sprintf("%s.pid", procName))
	cid := containerID(jobName, procName)

	return j.runcClient.RunContainer(
		pidFilePath,
		bundlePath,
		cid,
		stdout,
		stderr,
	)
}

// GetJob returns the following:
// - job, nil if the job is running (and no errors were encountered)
// - nil,nil if the job is not running and there is no other error
// - nil,error if there is any other error getting the job beyond it not running
func (j *RuncLifecycle) GetJob(jobName, procName string) (*models.Job, error) {
	cid := containerID(jobName, procName)
	container, err := j.runcClient.ContainerState(cid)
	if err != nil {
		return nil, err
	}

	if container == nil {
		return nil, nil
	}

	return &models.Job{
		Name:   container.ID,
		Pid:    container.Pid,
		Status: container.Status,
	}, nil
}

func (j *RuncLifecycle) OpenShell(jobName, procName string, stdin io.Reader, stdout, stderr io.Writer) error {
	cid := containerID(jobName, procName)
	return j.runcClient.Exec(cid, "/bin/bash", stdin, stdout, stderr)
}

func (j *RuncLifecycle) ListJobs() ([]models.Job, error) {
	containers, err := j.runcClient.ListContainers()
	if err != nil {
		return nil, err
	}

	var jobs []models.Job
	for _, c := range containers {
		job := models.Job{
			Name:   c.ID,
			Pid:    c.InitProcessPid,
			Status: c.Status,
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}

func (j *RuncLifecycle) StopJob(logger lager.Logger, jobName, procName string, exitTimeout time.Duration) error {
	cid := containerID(jobName, procName)

	err := j.runcClient.SignalContainer(cid, client.Term)
	if err != nil {
		return err
	}

	state, err := j.runcClient.ContainerState(cid)
	if err != nil {
		logger.Error("failed-to-fetch-state", err)
	} else {
		if state.Status == "stopped" {
			return nil
		}
	}

	timeout := j.clock.NewTimer(exitTimeout)
	stateTicker := j.clock.NewTicker(ContainerStatePollInterval)
	defer stateTicker.Stop()

	for {
		select {
		case <-stateTicker.C():
			state, err = j.runcClient.ContainerState(cid)
			if err != nil {
				logger.Error("failed-to-fetch-state", err)
			} else {
				if state.Status == "stopped" {
					return nil
				}
			}
		case <-timeout.C():
			err := j.runcClient.SignalContainer(cid, client.Quit)
			if err != nil {
				logger.Error("failed-to-sigquit", err)
			}

			j.clock.Sleep(ContainerSigQuitGracePeriod)
			return TimeoutError
		}
	}
}

func (j *RuncLifecycle) RemoveJob(jobName, procName string) error {
	cid := containerID(jobName, procName)

	err := j.runcClient.DeleteContainer(cid)
	if err != nil {
		return err
	}

	return j.runcClient.DestroyBundle(j.bundlePath(jobName, procName))
}

func (j *RuncLifecycle) bundlePath(jobName, procName string) string {
	return filepath.Join(j.systemRoot, "data", "bpm", "bundles", jobName, procName)
}

func containerID(jobName, procName string) string {
	if jobName == procName {
		return jobName
	}

	return fmt.Sprintf("%s.%s", jobName, procName)
}
