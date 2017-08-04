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
	"os"

	"github.com/spf13/cobra"
)

func init() {
	shellCommand.Flags().StringVarP(&processName, "process", "p", "", "The optional process name.")
	RootCmd.AddCommand(shellCommand)
}

var shellCommand = &cobra.Command{
	Long:    "start a shell inside the process container",
	RunE:    shell,
	Short:   "start a shell inside the process container",
	Use:     "shell <job-name>",
	PreRunE: shellPre,
}

func shellPre(cmd *cobra.Command, args []string) error {
	return validateInput(args)
}

func shell(cmd *cobra.Command, _ []string) error {
	silenceUsage(cmd)

	runcLifecycle := newRuncLifecycle()
	return runcLifecycle.OpenShell(bpmCfg, os.Stdin, cmd.OutOrStdout(), cmd.OutOrStderr())
}
