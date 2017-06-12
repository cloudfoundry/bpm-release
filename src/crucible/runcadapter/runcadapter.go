package runcadapter

import (
	"crucible/specbuilder"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

type RuncAdapter interface {
	BuildBundle(bundlesRoot, jobName string, jobSpec specs.Spec) (string, error)
	RunContainer(bundlePath, jobName string) error
	StopContainer(jobName string) error
	DestroyBundle(bundlesRoot, jobName string) error
}

type runcAdapter struct {
	runcPath     string
	userIdFinder specbuilder.UserIDFinder
}

func NewRuncAdapater(runcPath string, userIDFinder specbuilder.UserIDFinder) RuncAdapter {
	return &runcAdapter{
		runcPath:     runcPath,
		userIdFinder: userIDFinder,
	}
}
