// Copyright (C) 2018-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

	"github.com/spf13/cobra"

	"bpm/config"
	"bpm/exitstatus"
	"bpm/models"
	"bpm/runc/lifecycle"
)

var (
	// Volumes which come from command-line flags.
	volumes []string

	// Environment variables which come from command-line flags.
	env []string
)

func init() {
	runCommand.Flags().StringVarP(&procName, "process", "p", "", "the optional process name")
	runCommand.Flags().StringArrayVarP(&volumes, "volume", "v", []string{}, "Optional list of volumes (format: <path>[:<options>])")
	runCommand.Flags().StringArrayVarP(&env, "env", "e", []string{}, "Additional environment variables (format: KEY=VALUE")
	RootCmd.AddCommand(runCommand)
}

var runCommand = &cobra.Command{
	RunE:     run,
	Short:    "runs a bosh process synchronously",
	Use:      "run <job-name>",
	PreRunE:  runPre,
	PostRunE: runPost,
}

func runPre(cmd *cobra.Command, args []string) error {
	if err := validateInput(args); err != nil {
		return err
	}

	cmd.SilenceUsage = true

	if err := setupBpmLogs("run"); err != nil {
		return err
	}

	if err := mountRuncTmpfs(); err != nil {
		return err
	}

	return acquireLifecycleLock()
}

func runPost(cmd *cobra.Command, args []string) error {
	return releaseLifecycleLock()
}

func run(cmd *cobra.Command, _ []string) error {
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

	if err = procCfg.AddVolumes(volumes, bosh.Root(), bpmCfg.DefaultVolumes()); err != nil {
		logger.Error("invalid-volume-definition", err)
		return err
	}

	if err = procCfg.AddEnvVars(env, bosh.Root(), bpmCfg.DefaultVolumes()); err != nil {
		logger.Error("invalid-environment-definition", err)
		return err
	}

	runcLifecycle, err := newRuncLifecycle()
	if err != nil {
		return err
	}
	process, err := runcLifecycle.StatProcess(bpmCfg)
	if err != nil && !lifecycle.IsNotExist(err) {
		logger.Error("failed-getting-job", err)

		if cerr := forceCleanupBrokenRuncState(logger, runcLifecycle); cerr != nil {
			logger.Error("failed-cleaning-up-broken-job", cerr)
			return cerr
		}
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
		if err := runcLifecycle.RemoveProcess(logger, bpmCfg); err != nil {
			logger.Error("failed-to-cleanup", err)
			return fmt.Errorf("failed to clean up stale job-process: %s", err)
		}
		fallthrough
	default:
		if status, err := runcLifecycle.RunProcess(logger, bpmCfg, procCfg); err != nil {
			return &exitstatus.Error{
				Status: status,
				Err:    fmt.Errorf("failed to run job-process: %s", err),
			}
		}
	}

	return nil
}
