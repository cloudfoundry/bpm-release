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
	"os"
	"os/exec"
	"os/signal"

	"github.com/spf13/cobra"
)

var errLogs, allLogs bool

func init() {
	logsCommand.Flags().StringVarP(&procName, "process", "p", "", "The optional process name.")
	logsCommand.Flags().BoolVarP(&errLogs, "err", "e", false, "Tail error logs.")
	logsCommand.Flags().BoolVarP(&allLogs, "all", "a", false, "Tail all logs.")

	RootCmd.AddCommand(logsCommand)
}

var logsCommand = &cobra.Command{
	Long:    "Streams the logs for a given job",
	RunE:    logsForJob,
	Short:   "logs for job",
	Use:     "logs <job-name>",
	PreRunE: logsPre,
}

func logsPre(cmd *cobra.Command, args []string) error {
	return validateInput(args)
}

func logsForJob(cmd *cobra.Command, _ []string) error {
	var filesToTail []string

	if shouldTailStdout() {
		filesToTail = append(filesToTail, bpmCfg.Stdout())
	}

	if shouldTailStderr() {
		filesToTail = append(filesToTail, bpmCfg.Stderr())
	}

	if logsDontExist(filesToTail) {
		return errors.New("logs not found")
	}

	tailArgs := append([]string{"-f"}, filesToTail...)
	tailCmd := exec.Command("tail", tailArgs...)
	tailCmd.Stdout = cmd.OutOrStdout()
	tailCmd.Stderr = cmd.OutOrStderr()

	err := tailCmd.Start()
	if err != nil {
		return err
	}

	errCh := make(chan error)
	go func() {
		errCh <- tailCmd.Wait()
	}()

	signals := make(chan os.Signal)
	signal.Notify(signals)

	for {
		select {
		case sig := <-signals:
			tailCmd.Process.Signal(sig)
		case err := <-errCh:
			return err
		}
	}
}

func shouldTailStdout() bool {
	return !errLogs || allLogs
}

func shouldTailStderr() bool {
	return errLogs || allLogs
}

func logsDontExist(files []string) bool {
	for _, f := range files {
		_, err := os.Stat(f)
		if os.IsNotExist(err) {
			return true
		}
	}

	return false
}
