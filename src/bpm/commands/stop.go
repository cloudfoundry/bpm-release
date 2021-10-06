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
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"bpm/runc/lifecycle"
)

const DefaultStopTimeout = 15 * time.Second

func init() {
	stopCommand.Flags().StringVarP(&procName, "process", "p", "", "optional process name")
	RootCmd.AddCommand(stopCommand)
}

var stopCommand = &cobra.Command{
	RunE:     stop,
	Short:    "stops a BOSH Process",
	Use:      "stop <job-name>",
	PreRunE:  stopPre,
	PostRunE: stopPost,
}

func stopPre(cmd *cobra.Command, args []string) error {
	if err := validateInput(args); err != nil {
		return err
	}

	cmd.SilenceUsage = true

	if err := setupBpmLogs("stop"); err != nil {
		return err
	}

	return acquireLifecycleLock()
}

func stopPost(cmd *cobra.Command, args []string) error {
	return releaseLifecycleLock()
}

func stop(cmd *cobra.Command, _ []string) error {
	logger.Info("starting")
	defer logger.Info("complete")

	jobCfg, err := bpmCfg.ParseJobConfig()
	if err != nil {
		logger.Error("failed-to-parse-config", err)
		return fmt.Errorf("failed to parse job configuration: %s", err)
	}

	procCfg, err := processByNameFromJobConfig(jobCfg, procName)
	if err != nil {
		logger.Error("process-not-defined", err)
		return fmt.Errorf("process %q not present in job configuration (%s)", procName, bpmCfg.JobConfig())
	}

	runcLifecycle, err := newRuncLifecycle()
	if err != nil {
		return err
	}

	if _, err := runcLifecycle.StatProcess(bpmCfg); lifecycle.IsNotExist(err) {
		logger.Info("job-already-stopped")
		return nil
	} else if err != nil {
		logger.Error("failed-to-get-job", err)
		return fmt.Errorf("failed to get job-process status: %s", err)
	}

	if err := runcLifecycle.StopProcess(logger, bpmCfg, procCfg, DefaultStopTimeout); err != nil {
		logger.Error("failed-to-stop", err)
	}

	if err := runcLifecycle.RemoveProcess(logger, bpmCfg); err != nil {
		logger.Error("failed-to-cleanup", err)
		return fmt.Errorf("failed to cleanup job-process: %s", err)
	}

	return nil
}
