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
	"bpm/config"
	"bpm/models"
	"bpm/runc/lifecycle"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	startCommand.Flags().StringVarP(&procName, "process", "p", "", "The optional process name.")
	RootCmd.AddCommand(startCommand)
}

var startCommand = &cobra.Command{
	Long:     "Starts a BOSH Process",
	RunE:     start,
	Short:    "Starts a BOSH Process",
	Use:      "start <job-name>",
	PreRunE:  startPre,
	PostRunE: startPost,
}

func startPre(cmd *cobra.Command, args []string) error {
	if err := validateInput(args); err != nil {
		return err
	}

	cmd.SilenceUsage = true

	if err := setupBpmLogs("start"); err != nil {
		return err
	}

	return acquireLifecycleLock()
}

func startPost(cmd *cobra.Command, args []string) error {
	return releaseLifecycleLock()
}

func start(cmd *cobra.Command, _ []string) error {
	logger.Info("starting")
	defer logger.Info("complete")

	jobCfg, err := config.ParseJobConfig(bpmCfg.JobConfig())
	if err != nil {
		logger.Error("failed-to-parse-config", err)
		return fmt.Errorf("failed to parse job configuration: %s", err)
	}

	procCfg, err := processByNameFromJobConfig(jobCfg, procName)
	if err != nil {
		logger.Error("process-not-defined", err)
		return fmt.Errorf("process %q not present in job configuration (%s)", procName, bpmCfg.JobConfig())
	}

	runcLifecycle := newRuncLifecycle()
	process, err := runcLifecycle.StatProcess(bpmCfg)
	if err != nil && !lifecycle.IsNotExist(err) {
		logger.Error("failed-getting-job", err)
		return err
	}

	var state string
	if process != nil {
		state = process.Status
	}

	switch state {
	case models.ProcessStateRunning:
		logger.Info("process-already-running")
		return nil
	case models.ProcessStateFailed:
		logger.Info("removing-stopped-process")
		if err := runcLifecycle.RemoveProcess(bpmCfg); err != nil {
			logger.Error("failed-to-cleanup", err)
			return fmt.Errorf("failed to clean up stale job-process: %s", err)
		}
		fallthrough
	default:
		if err := runcLifecycle.StartProcess(logger, bpmCfg, procCfg); err != nil {
			return fmt.Errorf("failed to start job-process: %s", err)
		}
	}

	return nil
}
