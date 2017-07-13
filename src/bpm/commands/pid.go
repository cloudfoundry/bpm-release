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
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	pidCommand.Flags().StringVarP(&processName, "process", "p", "", "The optional process name.")
	RootCmd.AddCommand(pidCommand)
}

var pidCommand = &cobra.Command{
	Long:              "Displays the PID for a given job",
	RunE:              pidForJob,
	Short:             "PID for job",
	Use:               "pid <job-name>",
	PersistentPreRunE: pidPre,
}

func pidPre(cmd *cobra.Command, args []string) error {
	return validateInput(args)
}

func pidForJob(cmd *cobra.Command, _ []string) error {
	runcLifecycle := newRuncLifecycle()
	job, err := runcLifecycle.GetJob(jobName, processName)
	if err != nil {
		return fmt.Errorf("failed to get job: %s", err.Error())
	}

	if job.Pid <= 0 {
		return errors.New("no pid for job")
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%d\n", job.Pid)

	return nil
}
