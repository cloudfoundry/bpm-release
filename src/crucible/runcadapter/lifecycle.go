package runcadapter

import (
	"crucible/config"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/clock"
)

var TimeoutError = errors.New("failed to stop job within timeout")

type RuncJobLifecycle struct {
	clock        clock.Clock
	config       *config.CrucibleConfig
	jobName      string
	runcAdapter  RuncAdapter
	userIDFinder UserIDFinder
}

func NewRuncJobLifecycle(
	runcAdapter RuncAdapter,
	clock clock.Clock,
	jobName string,
	config *config.CrucibleConfig,
) *RuncJobLifecycle {
	return &RuncJobLifecycle{
		clock:       clock,
		config:      config,
		jobName:     jobName,
		runcAdapter: runcAdapter,
	}
}

func (j *RuncJobLifecycle) StartJob() error {
	pidDir, stdout, stderr, err := j.runcAdapter.CreateJobPrerequisites(config.BoshRoot(), j.jobName)
	if err != nil {
		return fmt.Errorf("failed to create system files: %s", err.Error())
	}
	defer stdout.Close()
	defer stderr.Close()

	spec, err := j.runcAdapter.BuildSpec(j.jobName, j.config)
	if err != nil {
		return err
	}

	bundlePath, err := j.runcAdapter.CreateBundle(config.BundlesRoot(), j.jobName, spec)
	if err != nil {
		return fmt.Errorf("bundle build failure: %s", err.Error())
	}

	return j.runcAdapter.RunContainer(pidDir, bundlePath, j.jobName, stdout, stderr)
}

func (j *RuncJobLifecycle) StopJob(exitTimeout time.Duration) error {
	err := j.runcAdapter.StopContainer(j.jobName)
	if err != nil {
		return err
	}

	state, err := j.runcAdapter.ContainerState(j.jobName)
	if err == nil {
		if state.Status == "stopped" {
			return nil
		}
	} else {
		// TODO: Log Here
	}

	stateTicker := j.clock.NewTicker(1 * time.Second)
	timeout := j.clock.NewTimer(exitTimeout)

	for {
		select {
		case <-stateTicker.C():
			state, err = j.runcAdapter.ContainerState(j.jobName)
			if err == nil {
				if state.Status == "stopped" {
					return nil
				}
			} else {
				// TODO: Log Here
			}
		case <-timeout.C():
			return TimeoutError
		}
	}
}

func (j *RuncJobLifecycle) RemoveJob() error {
	err := j.runcAdapter.DeleteContainer(j.jobName)
	if err != nil {
		return err
	}

	return j.runcAdapter.DestroyBundle(config.BundlesRoot(), j.jobName)
}
