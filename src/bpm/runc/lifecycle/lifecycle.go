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
	"bpm/runc/adapter"
	"bpm/runc/client"
	"bpm/usertools"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

var TimeoutError = errors.New("failed to stop job within timeout")

type RuncLifecycle struct {
	clock       clock.Clock
	runcClient  client.RuncClient
	runcAdapter adapter.RuncAdapter
	systemRoot  string
	userFinder  usertools.UserFinder
}

func NewRuncLifecycle(
	runcClient client.RuncClient,
	runcAdapter adapter.RuncAdapter,
	userFinder usertools.UserFinder,
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

func (j *RuncLifecycle) StartJob(jobName string, cfg *bpm.Config) error {
	user, err := j.userFinder.Lookup(usertools.VcapUser)
	if err != nil {
		return err
	}

	pidDir, stdout, stderr, err := j.runcAdapter.CreateJobPrerequisites(j.systemRoot, jobName, cfg, user)
	if err != nil {
		return fmt.Errorf("failed to create system files: %s", err.Error())
	}
	defer stdout.Close()
	defer stderr.Close()

	spec, err := j.runcAdapter.BuildSpec(j.systemRoot, jobName, cfg, user)
	if err != nil {
		return err
	}

	err = j.runcClient.CreateBundle(j.bundlePath(jobName, cfg), spec, user)
	if err != nil {
		return fmt.Errorf("bundle build failure: %s", err.Error())
	}

	pidFilePath := filepath.Join(pidDir, fmt.Sprintf("%s.pid", cfg.Name))
	cid := containerID(jobName, cfg.Name)

	return j.runcClient.RunContainer(
		pidFilePath,
		j.bundlePath(jobName, cfg),
		cid,
		stdout,
		stderr,
	)
}

func (j *RuncLifecycle) GetJob(jobName string, cfg *bpm.Config) (models.Job, error) {
	cid := containerID(jobName, cfg.Name)
	container, err := j.runcClient.ContainerState(cid)
	if err != nil {
		return models.Job{}, err
	}

	return models.Job{
		Name:   container.ID,
		Pid:    container.Pid,
		Status: container.Status,
	}, nil
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

func (j *RuncLifecycle) StopJob(logger lager.Logger, jobName string, cfg *bpm.Config, exitTimeout time.Duration) error {
	cid := containerID(jobName, cfg.Name)

	err := j.runcClient.StopContainer(cid)
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

	stateTicker := j.clock.NewTicker(1 * time.Second)
	timeout := j.clock.NewTimer(exitTimeout)

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
			return TimeoutError
		}
	}
}

func (j *RuncLifecycle) RemoveJob(jobName string, cfg *bpm.Config) error {
	cid := containerID(jobName, cfg.Name)

	err := j.runcClient.DeleteContainer(cid)
	if err != nil {
		return err
	}

	return j.runcClient.DestroyBundle(j.bundlePath(jobName, cfg))
}

func (j *RuncLifecycle) bundlePath(jobName string, cfg *bpm.Config) string {
	return filepath.Join(j.systemRoot, "data", "bpm", "bundles", jobName, cfg.Name)
}

func containerID(jobName, procName string) string {
	return fmt.Sprintf("%s-%s", jobName, procName)
}
