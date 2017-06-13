package lifecycle

import (
	"crucible/config"
	"crucible/runcadapter"
	"crucible/specbuilder"
	"fmt"
)

type JobLifecycle interface {
	StartJob() error
	StopJob() error
}

type runcJobLifecycle struct {
	runcAdapter     runcadapter.RuncAdapter
	bundlesRootPath string
	jobName         string
	config          *config.CrucibleConfig
	userIDFinder    specbuilder.UserIDFinder
}

func NewRuncJobLifecycle(
	runcPath string,
	bundlesRootPath string,
	jobName string,
	config *config.CrucibleConfig,
	userIDFinder specbuilder.UserIDFinder,
) *runcJobLifecycle {
	adapter := runcadapter.NewRuncAdapater(runcPath, userIDFinder)
	return &runcJobLifecycle{
		runcAdapter:     adapter,
		bundlesRootPath: bundlesRootPath,
		jobName:         jobName,
		config:          config,
		userIDFinder:    userIDFinder,
	}
}

func (j *runcJobLifecycle) StartJob() error {
	spec, err := specbuilder.Build(j.jobName, j.config, j.userIDFinder)
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

	return j.runcAdapter.DestroyBundle(j.bundlesRootPath, j.jobName)
}
