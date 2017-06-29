package runcadapter

import (
	"crucible/config"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

const VCAP_USER = "vcap"

var TimeoutError = errors.New("failed to stop job within timeout")

type RuncJobLifecycle struct {
	clock        clock.Clock
	cfg          *config.CrucibleConfig
	jobName      string
	runcClient   RuncClient
	runcAdapter  RuncAdapter
	systemRoot   string
	userIDFinder UserIDFinder
}

func NewRuncJobLifecycle(
	runcClient RuncClient,
	runcAdapter RuncAdapter,
	userIDFinder UserIDFinder,
	clock clock.Clock,
	systemRoot string,
	jobName string,
	cfg *config.CrucibleConfig,
) *RuncJobLifecycle {
	return &RuncJobLifecycle{
		clock:        clock,
		cfg:          cfg,
		jobName:      jobName,
		runcClient:   runcClient,
		runcAdapter:  runcAdapter,
		systemRoot:   systemRoot,
		userIDFinder: userIDFinder,
	}
}

func (j *RuncJobLifecycle) StartJob() error {
	user, err := j.userIDFinder.Lookup(VCAP_USER)
	if err != nil {
		return err
	}

	pidDir, stdout, stderr, err := j.runcAdapter.CreateJobPrerequisites(j.systemRoot, j.jobName, j.cfg, user)
	if err != nil {
		return fmt.Errorf("failed to create system files: %s", err.Error())
	}
	defer stdout.Close()
	defer stderr.Close()

	spec, err := j.runcAdapter.BuildSpec(j.systemRoot, j.jobName, j.cfg, user)
	if err != nil {
		return err
	}

	err = j.runcClient.CreateBundle(j.bundlePath(), spec, user)
	if err != nil {
		return fmt.Errorf("bundle build failure: %s", err.Error())
	}

	pidFilePath := filepath.Join(pidDir, fmt.Sprintf("%s.pid", j.cfg.Name))
	cid := containerID(j.jobName, j.cfg.Name)

	return j.runcClient.RunContainer(
		pidFilePath,
		j.bundlePath(),
		cid,
		stdout,
		stderr,
	)
}

func (j *RuncJobLifecycle) StopJob(logger lager.Logger, exitTimeout time.Duration) error {
	cid := containerID(j.jobName, j.cfg.Name)

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

func (j *RuncJobLifecycle) RemoveJob() error {
	cid := containerID(j.jobName, j.cfg.Name)

	err := j.runcClient.DeleteContainer(cid)
	if err != nil {
		return err
	}

	return j.runcClient.DestroyBundle(j.bundlePath())
}

func containerID(jobName, procName string) string {
	return fmt.Sprintf("%s-%s", jobName, procName)
}

func (j *RuncJobLifecycle) bundlePath() string {
	return filepath.Join(j.systemRoot, "data", "crucible", "bundles", j.jobName, j.cfg.Name)
}
