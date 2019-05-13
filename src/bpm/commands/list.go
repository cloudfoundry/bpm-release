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
	"os"

	"github.com/spf13/cobra"

	"bpm/config"
	"bpm/jobid"
	"bpm/models"
	"bpm/presenters"
)

func init() {
	RootCmd.AddCommand(listCommandCommand)
}

var listCommandCommand = &cobra.Command{
	RunE:  listContainers,
	Short: "list the state of bpm containers",
	Use:   "list",
}

func listContainers(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	processes := []*models.Process{}
	for _, job := range boshEnv.JobNames() {
		bpmCfg := config.NewBPMConfig(boshEnv, job, "")
		jobCfg, err := bpmCfg.ParseJobConfig()
		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "invalid config for %s: %s", job, err.Error())
			continue
		}

		for _, process := range jobCfg.Processes {
			procCfg := config.NewBPMConfig(boshEnv, job, process.Name)
			processes = append(processes, &models.Process{
				Name:   procCfg.ContainerID(),
				Status: models.ProcessStateStopped,
			})
		}
	}

	runcLifecycle, err := newRuncLifecycle()
	if err != nil {
		return err
	}
	runningProcesses, err := runcLifecycle.ListProcesses()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "failed to list jobs: %s\n", err.Error())
		return err
	}

	for _, process := range runningProcesses {
		processes, err = updateProcess(processes, process)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "extra process running: %s", err.Error())
		}
	}

	err = presenters.PrintJobs(processes, cmd.OutOrStdout())
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "failed to display jobs: %s\n", err.Error())
		return err
	}

	return nil
}

func updateProcess(processes []*models.Process, process *models.Process) ([]*models.Process, error) {
	for i := range processes {
		if processes[i].Name == process.Name {
			processes[i] = process
			return processes, nil
		}
	}

	decodedName, err := jobid.Decode(process.Name)
	if err != nil {
		return processes, err
	}
	return processes, fmt.Errorf("process (%s) not defined", decodedName)
}
