package runcadapter

import specs "github.com/opencontainers/runtime-spec/specs-go"

type RuncAdapater interface {
	BuildBundle(bundleRoot, jobName string, jobSpec specs.Spec) (string, error)
	RunContainer(bundlePath, jobName string) error
}

type runcAdapter struct {
	runcPath string
}

func NewRuncAdapater(runcPath string) *runcAdapter {
	return &runcAdapter{
		runcPath: runcPath,
	}
}
