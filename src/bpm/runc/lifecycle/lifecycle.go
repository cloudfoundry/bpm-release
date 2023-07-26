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

package lifecycle

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager/v3"

	"bpm/config"
	"bpm/models"
	"bpm/runc/client"
	"bpm/usertools"
)

const (
	ContainerSigQuitGracePeriod = 2 * time.Second
	ContainerStatePollInterval  = 1 * time.Second

	ContainerStateRunning = "running"
	ContainerStatePaused  = "paused"
	ContainerStateStopped = "stopped"
)

var (
	timeoutError    = errors.New("failed to stop job within timeout")
	isNotExistError = errors.New("process is not running or could not be found")
)

func IsNotExist(err error) bool {
	return err == isNotExistError
}

//go:generate go run -mod=vendor github.com/golang/mock/mockgen -copyright_file ./mock_lifecycle/header.txt -destination ./mock_lifecycle/mocks.go bpm/runc/lifecycle UserFinder,CommandRunner,RuncAdapter,RuncClient

type UserFinder interface {
	Lookup(username string) (specs.User, error)
}

type CommandRunner interface {
	Run(*exec.Cmd) error
}

type RuncAdapter interface {
	CreateJobPrerequisites(bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig, user specs.User) (*os.File, *os.File, error)
	BuildSpec(logger lager.Logger, bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig, user specs.User) (specs.Spec, error)
}

type RuncClient interface {
	CreateBundle(bundlePath string, jobSpec specs.Spec, user specs.User) error
	RunContainer(pidFilePath, bundlePath, containerID string, detach bool, stdout, stderr io.Writer) (int, error)
	Exec(containerID, command string, stdin io.Reader, stdout, stderr io.Writer) error
	ContainerState(containerID string) (*specs.State, error)
	ListContainers() ([]client.ContainerState, error)
	SignalContainer(containerID string, signal client.Signal) error
	DeleteContainer(containerID string) error
	DestroyBundle(bundlePath string) error
}

type RuncLifecycle struct {
	clock         clock.Clock
	commandRunner CommandRunner
	runcAdapter   RuncAdapter
	runcClient    RuncClient
	userFinder    UserFinder
	deleteFile    func(string) error
}

func NewRuncLifecycle(
	runcClient RuncClient,
	runcAdapter RuncAdapter,
	userFinder UserFinder,
	commandRunner CommandRunner,
	clock clock.Clock,
	deleteFile func(string) error,
) *RuncLifecycle {
	return &RuncLifecycle{
		clock:         clock,
		runcClient:    runcClient,
		runcAdapter:   runcAdapter,
		userFinder:    userFinder,
		commandRunner: commandRunner,
		deleteFile:    deleteFile,
	}
}

func (j *RuncLifecycle) StartProcess(logger lager.Logger, bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig) error {
	logger = logger.Session("start-process")
	logger.Info("starting")
	defer logger.Info("complete")

	stdout, stderr, err := j.setupProcess(logger, bpmCfg, procCfg)
	if err != nil {
		return err
	}
	defer stdout.Close()
	defer stderr.Close()

	logger.Info("running-container")
	_, err = j.runcClient.RunContainer(
		bpmCfg.PidFile().External(),
		bpmCfg.BundlePath(),
		bpmCfg.ContainerID(),
		true,
		stdout,
		stderr,
	)

	return err
}

func (j *RuncLifecycle) RunProcess(logger lager.Logger, bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig) (int, error) {
	logger = logger.Session("run-process")
	logger.Info("starting")
	defer logger.Info("complete")

	stdout, stderr, err := j.setupProcess(logger, bpmCfg, procCfg)
	if err != nil {
		return 0, err
	}
	defer stdout.Close()
	defer stderr.Close()

	logger.Info("running-container")
	return j.runcClient.RunContainer(
		bpmCfg.PidFile().External(),
		bpmCfg.BundlePath(),
		bpmCfg.ContainerID(),
		false,
		io.MultiWriter(stdout, os.Stdout),
		io.MultiWriter(stderr, os.Stderr),
	)
}

func (j *RuncLifecycle) setupProcess(logger lager.Logger, bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig) (io.WriteCloser, io.WriteCloser, error) {
	user, err := j.userFinder.Lookup(usertools.VcapUser)
	if err != nil {
		return nil, nil, err
	}

	logger.Info("creating-job-prerequisites")
	stdout, stderr, err := j.runcAdapter.CreateJobPrerequisites(bpmCfg, procCfg, user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create system files: %s", err.Error())
	}

	logger.Info("building-spec")
	spec, err := j.runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
	if err != nil {
		return nil, nil, err
	}

	logger.Info("creating-bundle")
	err = j.runcClient.CreateBundle(bpmCfg.BundlePath(), spec, user)
	if err != nil {
		return nil, nil, fmt.Errorf("bundle build failure: %s", err.Error())
	}

	if procCfg.Hooks != nil {
		preStartCmd := exec.Command(procCfg.Hooks.PreStart)
		preStartCmd.Env = spec.Process.Env
		preStartCmd.Stdout = stdout
		preStartCmd.Stderr = stderr

		err := j.commandRunner.Run(preStartCmd)
		if err != nil {
			return nil, nil, fmt.Errorf("prestart hook failed: %s", err.Error())
		}
	}

	return stdout, stderr, nil
}

func (j *RuncLifecycle) StatProcess(cfg *config.BPMConfig) (*models.Process, error) {
	container, err := j.runcClient.ContainerState(cfg.ContainerID())
	if err != nil {
		return nil, err
	}

	if container == nil {
		return nil, isNotExistError
	}

	return newProcessFromContainerState(
		container.ID,
		container.Status,
		container.Pid,
	), nil
}

func (j *RuncLifecycle) OpenShell(cfg *config.BPMConfig, stdin io.Reader, stdout, stderr io.Writer) error {
	return j.runcClient.Exec(cfg.ContainerID(), "/bin/bash", stdin, stdout, stderr)
}

func (j *RuncLifecycle) ListProcesses() ([]*models.Process, error) {
	containers, err := j.runcClient.ListContainers()
	if err != nil {
		return nil, err
	}

	var processes []*models.Process
	for _, c := range containers {
		processes = append(processes, newProcessFromContainerState(
			c.ID,
			containerStateFromString(c.Status),
			c.InitProcessPid,
		))
	}

	return processes, nil
}

func (j *RuncLifecycle) StopProcess(logger lager.Logger, cfg *config.BPMConfig, procCfg *config.ProcessConfig, exitTimeout time.Duration) error {
	err := j.runcClient.SignalContainer(cfg.ContainerID(), procCfg.ParseShutdownSignal())
	if err != nil {
		return err
	}

	state, err := j.runcClient.ContainerState(cfg.ContainerID())
	if err != nil {
		logger.Error("failed-to-fetch-state", err)
	} else {
		if state.Status == ContainerStateStopped {
			return nil
		}
	}

	timeout := j.clock.NewTimer(exitTimeout)
	stateTicker := j.clock.NewTicker(ContainerStatePollInterval)
	defer stateTicker.Stop()

	for {
		select {
		case <-stateTicker.C():
			state, err = j.runcClient.ContainerState(cfg.ContainerID())
			if err != nil {
				logger.Error("failed-to-fetch-state", err)
			} else {
				if state.Status == ContainerStateStopped {
					return nil
				}
			}
		case <-timeout.C():
			err := j.runcClient.SignalContainer(cfg.ContainerID(), client.Quit)
			if err != nil {
				logger.Error("failed-to-sigquit", err)
			}

			j.clock.Sleep(ContainerSigQuitGracePeriod)
			return timeoutError
		}
	}
}

func (j *RuncLifecycle) RemoveProcess(logger lager.Logger, cfg *config.BPMConfig) error {
	logger.Info("forcefully-deleting-container")
	if err := j.runcClient.DeleteContainer(cfg.ContainerID()); err != nil {
		return err
	}

	logger.Info("destroying-bundle")
	if err := j.runcClient.DestroyBundle(cfg.BundlePath()); err != nil {
		return err
	}

	logger.Info("deleting-pidfile")
	return j.deleteFile(cfg.PidFile().External())
}

func newProcessFromContainerState(id string, status specs.ContainerState, pid int) *models.Process {
	return &models.Process{
		Name:   id,
		Pid:    pid,
		Status: containerStateToString(status),
	}
}

type commandRunner struct{}

func NewCommandRunner() CommandRunner          { return &commandRunner{} }
func (*commandRunner) Run(cmd *exec.Cmd) error { return cmd.Run() }

func containerStateToString(cs specs.ContainerState) string {
	switch cs {
	case specs.StateCreating:
		return models.ProcessStateCreating
	case specs.StateCreated:
		return models.ProcessStateCreated
	case specs.StateRunning:
		return models.ProcessStateRunning
	case specs.StateStopped:
		return models.ProcessStateFailed
	default:
		return models.ProcessStateFailed
	}
}

func containerStateFromString(status string) specs.ContainerState {
	switch status {
	case models.ProcessStateCreating:
		return specs.StateCreating
	case models.ProcessStateCreated:
		return specs.StateCreated
	case models.ProcessStateRunning:
		return specs.StateRunning
	case models.ProcessStateFailed:
		return specs.StateStopped
	default:
		return models.ProcessStateFailed
	}
}
