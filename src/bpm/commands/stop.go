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
	"time"

	"github.com/spf13/cobra"
)

const DefaultStopTimeout = 20 * time.Second

func init() {
	stopCommand.Flags().StringVarP(&procName, "process", "p", "", "The optional process name.")
	RootCmd.AddCommand(stopCommand)
}

var stopCommand = &cobra.Command{
	Long:     "Stops a BOSH Process",
	RunE:     stop,
	Short:    "Stops a BOSH Process",
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

	runcLifecycle := newRuncLifecycle()
	job, err := runcLifecycle.GetProcess(bpmCfg)
	if err != nil {
		logger.Error("failed-to-get-job", err)
		return err
	}

	if job == nil {
		logger.Info("job-already-stopped")
		return nil
	}

	err = runcLifecycle.StopProcess(logger, bpmCfg, DefaultStopTimeout)
	if err != nil {
		logger.Error("failed-to-stop", err)
	}

	return runcLifecycle.RemoveProcess(bpmCfg)
}
