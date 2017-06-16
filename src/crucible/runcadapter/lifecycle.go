package runcadapter

import (
	"crucible/config"
	"fmt"
)

type runcJobLifecycle struct {
	runcAdapter  RuncAdapter
	jobName      string
	config       *config.CrucibleConfig
	userIDFinder UserIDFinder
}

func NewRuncJobLifecycle(
	runcAdapter RuncAdapter,
	jobName string,
	config *config.CrucibleConfig,
) *runcJobLifecycle {
	return &runcJobLifecycle{
		runcAdapter: runcAdapter,
		jobName:     jobName,
		config:      config,
	}
}

func (j *runcJobLifecycle) StartJob() error {
	spec, err := j.runcAdapter.BuildSpec(j.jobName, j.config)
	if err != nil {
		return err
	}

	bundlePath, err := j.runcAdapter.BuildBundle(config.BundlesRoot(), j.jobName, spec)
	if err != nil {
		return fmt.Errorf("bundle build failure: %s", err.Error())
	}

	return j.runcAdapter.RunContainer(bundlePath, j.jobName)
}

func (j *runcJobLifecycle) StopJob() error {
	err := j.runcAdapter.StopContainer(j.jobName)
	if err != nil {
		// TODO: test me?
		return err
	}

	return j.runcAdapter.DestroyBundle(config.BundlesRoot(), j.jobName)
}
