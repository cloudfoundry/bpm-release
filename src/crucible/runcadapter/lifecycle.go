package runcadapter

import (
	"crucible/config"
	"fmt"
)

type RuncJobLifecycle struct {
	runcAdapter  RuncAdapter
	jobName      string
	config       *config.CrucibleConfig
	userIDFinder UserIDFinder
}

func NewRuncJobLifecycle(
	runcAdapter RuncAdapter,
	jobName string,
	config *config.CrucibleConfig,
) *RuncJobLifecycle {
	return &RuncJobLifecycle{
		runcAdapter: runcAdapter,
		jobName:     jobName,
		config:      config,
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

func (j *RuncJobLifecycle) StopJob() error {
	err := j.runcAdapter.StopContainer(j.jobName)
	if err != nil {
		return err
	}

	return j.runcAdapter.DestroyBundle(config.BundlesRoot(), j.jobName)
}
