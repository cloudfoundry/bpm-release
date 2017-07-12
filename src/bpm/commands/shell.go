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
	"os"

	"github.com/spf13/cobra"
)

func init() {
	shellCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	shellCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the bpm configuration file.")
	RootCmd.AddCommand(shellCommand)
}

var shellCommand = &cobra.Command{
	Long:              "start a shell inside the process container",
	RunE:              shell,
	Short:             "start a shell inside the process container",
	Use:               "shell",
	PersistentPreRunE: shellPre,
}

func shellPre(cmd *cobra.Command, _ []string) error {
	return validateJobandConfigFlags()
}

func shell(cmd *cobra.Command, _ []string) error {
	cfg, err := bpm.ParseConfig(configPath)
	if err != nil {
		return err
	}

	runcLifecycle := newRuncLifecycle()
	return runcLifecycle.OpenShell(jobName, cfg, os.Stdin, cmd.OutOrStdout(), cmd.OutOrStderr())
}
