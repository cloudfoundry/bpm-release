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
	"errors"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/spf13/cobra"
)

const DefaultStopTimeout = 20 * time.Second

func init() {
	stopCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	stopCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the bpm configuration file.")
	RootCmd.AddCommand(stopCommand)
}

var stopCommand = &cobra.Command{
	Long:    "Stops a BOSH Process",
	RunE:    stop,
	Short:   "Stops a BOSH Process",
	Use:     "stop <job-name>",
	PreRunE: stopPre,
}

func stopPre(cmd *cobra.Command, _ []string) error {
	if err := validateStopFlags(jobName, configPath); err != nil {
		return err
	}

	return setupBpmLogs()
}

func stop(cmd *cobra.Command, _ []string) error {
	cfg, err := bpm.ParseConfig(configPath)
	if err != nil {
		logger.Error("failed-to-parse-config", err)
		return err
	}

	logger = logger.Session("stop", lager.Data{"process": cfg.Name})
	logger.Info("starting")
	defer logger.Info("complete")

	runcLifecycle := newRuncLifecycle()
	err = runcLifecycle.StopJob(logger, jobName, cfg, DefaultStopTimeout)
	if err != nil {
		logger.Error("failed-to-stop", err)
	}

	return runcLifecycle.RemoveJob(jobName, cfg)
}

func validateStopFlags(jobName, configPath string) error {
	if jobName == "" {
		return errors.New("must specify a job")
	}

	if configPath == "" {
		return errors.New("must specify a configuration file")
	}

	return nil
}
