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
	"bpm/config"
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
		return err
	}

	procCfg, err := processByNameFromJobConfig(jobCfg, procName)
	if err != nil {
		err = fmt.Errorf("invalid-process: %s", procName)
		logger.Error("process-not-defined", err)
		return err
	}

	runcLifecycle := newRuncLifecycle()
	job, err := runcLifecycle.GetProcess(bpmCfg)
	if err != nil {
		logger.Error("failed-getting-job", err)
	}

	var state string
	if job != nil {
		state = job.Status
	}

	switch state {
	case lifecycle.ContainerStateRunning:
		logger.Info("process-already-running")
		return nil
	case lifecycle.ContainerStateStopped:
		logger.Info("removing-stopped-process")
		removeErr := runcLifecycle.RemoveProcess(bpmCfg)
		if removeErr != nil {
			logger.Error("failed-to-cleanup", removeErr)
			return removeErr
		}
		fallthrough
	default:
		return runcLifecycle.StartProcess(logger, bpmCfg, procCfg)
	}
}
