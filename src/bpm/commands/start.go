// Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.

// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License”);
// you may not use this file except in compliance with the License.

// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
// License for the specific language governing permissions and limitations
// under the License.

package commands

import (
	"bpm/bpm"
	"errors"

	"code.cloudfoundry.org/lager"

	"github.com/spf13/cobra"
)

func init() {
	startCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	startCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the bpm configuration file.")
	RootCmd.AddCommand(startCommand)
}

var startCommand = &cobra.Command{
	Long:              "Starts a BOSH Process",
	RunE:              start,
	Short:             "Starts a BOSH Process",
	Use:               "start <job-name>",
	PersistentPreRunE: startPre,
}

func startPre(cmd *cobra.Command, _ []string) error {
	if err := validateStartFlags(jobName, configPath); err != nil {
		return err
	}

	return setupBpmLogs()
}

func start(cmd *cobra.Command, _ []string) error {
	cfg, err := bpm.ParseConfig(configPath)
	if err != nil {
		logger.Error("failed-to-parse-config", err)
		return err
	}

	logger = logger.Session("start", lager.Data{"process": cfg.Name})
	logger.Info("starting")
	defer logger.Info("complete")

	runcLifecycle := newRuncLifecycle()
	err = runcLifecycle.StartJob(jobName, cfg)
	if err != nil {
		logger.Error("failed-to-start", err)

		removeErr := runcLifecycle.RemoveJob(jobName, cfg)
		if removeErr != nil {
			logger.Error("failed-to-cleanup", removeErr)
		}

		return err
	}

	return nil
}

func validateStartFlags(jobName, configPath string) error {
	if jobName == "" {
		return errors.New("must specify a job")
	}

	if configPath == "" {
		return errors.New("must specify a configuration file")
	}

	return nil
}
