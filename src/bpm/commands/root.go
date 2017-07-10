// Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
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
	"bpm/bpm"
	"bpm/runc/adapter"
	"bpm/runc/client"
	"bpm/runc/lifecycle"
	"bpm/usertools"
	"errors"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"

	"github.com/spf13/cobra"
)

var jobName, configPath string
var logger lager.Logger

var userFinder = usertools.NewUserFinder()

var RootCmd = &cobra.Command{
	Long:          "A bosh process manager for starting and stopping release jobs",
	RunE:          root,
	Short:         "A bosh process manager for starting and stopping release jobs",
	SilenceErrors: true,
	Use:           "bpm",
	ValidArgs:     []string{"start", "stop", "list"},
}

func root(cmd *cobra.Command, args []string) error {
	return errors.New("Exit code 1")
}

func validateJobandConfigFlags() error {
	if jobName == "" {
		return errors.New("must specify a job")
	}

	if configPath == "" {
		return errors.New("must specify a configuration file")
	}

	return nil
}

func setupBpmLogs() error {
	bpmLogFileLocation := filepath.Join(bpm.BoshRoot(), "sys", "log", jobName, "bpm.log")
	err := os.MkdirAll(filepath.Join(bpm.BoshRoot(), "sys", "log", jobName), 0750)
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(bpmLogFileLocation, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0700)
	if err != nil {
		return err
	}

	usr, err := userFinder.Lookup(usertools.VcapUser)
	if err != nil {
		return err
	}

	err = os.Chown(bpmLogFileLocation, int(usr.UID), int(usr.GID))
	if err != nil {
		return err
	}

	logger, _ = lagerflags.NewFromConfig("bpm", lagerflags.DefaultLagerConfig())
	logger.RegisterSink(lager.NewWriterSink(logFile, lager.INFO))
	logger = logger.WithData(lager.Data{"job": jobName})

	return nil
}

func newRuncLifecycle() *lifecycle.RuncLifecycle {
	runcClient := client.NewRuncClient(bpm.RuncPath(), bpm.RuncRoot())
	runcAdapter := adapter.NewRuncAdapter()
	clock := clock.NewClock()

	return lifecycle.NewRuncLifecycle(
		runcClient,
		runcAdapter,
		userFinder,
		clock,
		bpm.BoshRoot(),
	)
}
