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

package commands

import (
	"errors"
	"fmt"
	"os"
	"os/user"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"

	"bpm/cgroups"
	"bpm/config"
	"bpm/mount"
	"bpm/runc/adapter"
	"bpm/runc/client"
	"bpm/runc/lifecycle"
	"bpm/sysfeat"
	"bpm/usertools"
)

var (
	bpmCfg      *config.BPMConfig
	logger      lager.Logger
	procName    string
	showVersion bool
)

var userFinder = usertools.NewUserFinder()
var bosh = config.NewBosh(os.Getenv("BPM_BOSH_ROOT"))

func init() {
	RootCmd.PersistentFlags().BoolVar(&showVersion, "version", false, "print BPM version")
}

var RootCmd = &cobra.Command{
	Long:              "A bosh process manager for starting and stopping release jobs",
	RunE:              root,
	Short:             "A bosh process manager for starting and stopping release jobs",
	SilenceErrors:     true,
	Use:               "bpm",
	PersistentPreRunE: rootPre,
}

func rootPre(cmd *cobra.Command, _ []string) error {
	if showVersion {
		version(cmd, []string{})
		os.Exit(0)
	}

	usr, err := user.Current()
	if err != nil {
		return err
	}

	if usr.Uid != "0" && usr.Gid != "0" {
		cmd.SilenceUsage = true
		return errors.New("bpm must be run as root. Please run 'sudo -i' to become the root user.")
	}

	return cgroups.Setup()
}

func root(cmd *cobra.Command, args []string) error {
	return errors.New("Exit code 1")
}

func validateInput(args []string) error {
	if len(args) < 1 {
		return errors.New("must specify a job")
	}

	jobName := args[0]

	if procName == "" {
		procName = jobName
	}

	bpmCfg = config.NewBPMConfig(bosh.Root(), jobName, procName)

	return nil
}

func setupBpmLogs(sessionName string) error {
	err := os.MkdirAll(bpmCfg.LogDir(), 0750)
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(bpmCfg.BPMLog(), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	usr, err := userFinder.Lookup(usertools.VcapUser)
	if err != nil {
		return err
	}

	err = os.Chown(bpmCfg.BPMLog(), int(usr.UID), int(usr.GID))
	if err != nil {
		return err
	}

	logger = lager.NewLogger("bpm")
	logger.RegisterSink(lager.NewPrettySink(logFile, lager.INFO))
	logger = logger.Session(sessionName, lager.Data{
		"job":     bpmCfg.JobName(),
		"process": bpmCfg.ProcName(),
	})

	return nil
}

func acquireLifecycleLock() error {
	l := logger.Session("acquiring-lifecycle-lock")
	l.Info("starting")
	defer l.Info("complete")

	err := os.MkdirAll(bpmCfg.PidDir(), 0700)
	if err != nil {
		l.Error("failed-to-create-lock-dir", err)
		return err
	}

	f, err := os.OpenFile(bpmCfg.LockFile(), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		l.Error("failed-to-create-lock-file", err)
		return err
	}

	err = unix.Flock(int(f.Fd()), unix.LOCK_EX)
	if err != nil {
		l.Error("failed-to-acquire-lock", err)
		return err
	}

	return nil
}

func releaseLifecycleLock() error {
	l := logger.Session("releasing-lifecycle-lock")
	l.Info("starting")
	defer l.Info("complete")

	err := os.RemoveAll(bpmCfg.LockFile())
	if err != nil {
		l.Error("failed-to-remove-lock-file", err)
		return err
	}

	return nil
}

func newRuncLifecycle() (*lifecycle.RuncLifecycle, error) {
	runcClient := client.NewRuncClient(
		config.RuncPath(bosh.Root()),
		config.RuncRoot(bosh.Root()),
	)
	features, err := sysfeat.Fetch()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch system features: %q", err)
	}
	runcAdapter := adapter.NewRuncAdapter(*features, mount.MakeShared)
	clock := clock.NewClock()

	return lifecycle.NewRuncLifecycle(
		runcClient,
		runcAdapter,
		userFinder,
		lifecycle.NewCommandRunner(),
		clock,
		os.Remove,
	), nil
}

func processByNameFromJobConfig(jobCfg *config.JobConfig, procName string) (*config.ProcessConfig, error) {
	for _, processConfig := range jobCfg.Processes {
		if processConfig.Name == procName {
			return processConfig, nil
		}
	}

	return nil, fmt.Errorf("invalid process: %s", procName)
}
