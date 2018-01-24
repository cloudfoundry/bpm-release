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
	"bpm/presenters"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(listCommandCommand)
}

var listCommandCommand = &cobra.Command{
	Long:  "Lists the state of bpm containers",
	RunE:  listContainers,
	Short: "List containers",
	Use:   "list",
}

func listContainers(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	runcLifecycle := newRuncLifecycle()
	jobs, err := runcLifecycle.ListProcesses()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "failed to list jobs: %s\n", err.Error())
		return err
	}

	err = presenters.PrintJobs(jobs, cmd.OutOrStdout())
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "failed to display jobs: %s\n", err.Error())
		return err
	}

	return nil
}
