package runcadapter

import (
	"crucible/config"
	"crucible/models"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
)

const VcapUser = "vcap"

var TimeoutError = errors.New("failed to stop job within timeout")

type RuncJobLifecycle struct {
	clock        clock.Clock
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
) *RuncJobLifecycle {
	return &RuncJobLifecycle{
		clock:        clock,
		runcClient:   runcClient,
		runcAdapter:  runcAdapter,
		systemRoot:   systemRoot,
		userIDFinder: userIDFinder,
	}
}

func (j *RuncJobLifecycle) StartJob(jobName string, cfg *config.CrucibleConfig) error {
	user, err := j.userIDFinder.Lookup(VcapUser)
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

func (j *RuncJobLifecycle) ListJobs() ([]models.Job, error) {
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

func (j *RuncJobLifecycle) StopJob(logger lager.Logger, jobName string, cfg *config.CrucibleConfig, exitTimeout time.Duration) error {
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

func (j *RuncJobLifecycle) RemoveJob(jobName string, cfg *config.CrucibleConfig) error {
	cid := containerID(jobName, cfg.Name)

	err := j.runcClient.DeleteContainer(cid)
	if err != nil {
		return err
	}

	return j.runcClient.DestroyBundle(j.bundlePath(jobName, cfg))
}

func (j *RuncJobLifecycle) bundlePath(jobName string, cfg *config.CrucibleConfig) string {
	return filepath.Join(j.systemRoot, "data", "crucible", "bundles", jobName, cfg.Name)
}

func containerID(jobName, procName string) string {
	return fmt.Sprintf("%s-%s", jobName, procName)
}
