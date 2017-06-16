package runcadapter

import (
	"crucible/config"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

//go:generate counterfeiter . RuncAdapter
type RuncAdapter interface {
	BuildSpec(jobName string, jobConfig *config.CrucibleConfig) (specs.Spec, error)
	BuildBundle(bundlesRoot, jobName string, jobSpec specs.Spec) (string, error)
	RunContainer(bundlePath, jobName string) error
	StopContainer(jobName string) error
	DestroyBundle(bundlesRoot, jobName string) error
}

type runcAdapter struct {
	runcPath     string
	userIDFinder UserIDFinder
}

func NewRuncAdapter(runcPath string, userIDFinder UserIDFinder) RuncAdapter {
	return &runcAdapter{
		runcPath:     runcPath,
		userIDFinder: userIDFinder,
	}
}

func (a *runcAdapter) RunContainer(bundlePath, jobName string) error {
	cruciblePidDir := filepath.Join(config.BoshRoot(), "sys", "run", "crucible")
	err := os.MkdirAll(cruciblePidDir, 0700)
	if err != nil {
		// Test Me
		return err
	}

	runcCmd := exec.Command(
		a.runcPath,
		"run",
		"--bundle", bundlePath,
		"--pid-file", filepath.Join(cruciblePidDir, fmt.Sprintf("%s.pid", jobName)),
		"--detach",
		jobName,
	)

	jobLogDir := filepath.Join(config.BoshRoot(), "sys", "log", jobName)
	err = os.MkdirAll(jobLogDir, 0700)
	if err != nil {
		// Test Me
		return err
	}

	stdoutFileLocation := fmt.Sprintf("%s/%s.out.log", jobLogDir, jobName)
	stderrFileLocation := fmt.Sprintf("%s/%s.err.log", jobLogDir, jobName)

	stdout, err := os.Create(stdoutFileLocation)
	if err != nil {
		// Test Me
		return err
	}

	stderr, err := os.Create(stderrFileLocation)
	if err != nil {
		// Test Me
		return err
	}

	runcCmd.Stdout = stdout
	runcCmd.Stderr = stderr

	err = runcCmd.Start()
	if err != nil {
		return err
	}

	return runcCmd.Wait()
}

func (a *runcAdapter) StopContainer(jobName string) error {
	runcCmd := exec.Command(
		a.runcPath,
		"delete",
		"-f",
		jobName,
	)

	err := runcCmd.Start()
	if err != nil {
		return err
	}

	return runcCmd.Wait()
}
