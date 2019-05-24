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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/spf13/cobra"
)

var (
	errLogs,
	allLogs,
	follow,
	quiet bool

	numLines int
)

func init() {
	logsCommand.Flags().BoolVarP(&allLogs, "all", "a", false, "show both stdout and stderr")
	logsCommand.Flags().BoolVarP(&errLogs, "err", "e", false, "show stderr")
	logsCommand.Flags().BoolVarP(&follow, "follow", "f", false, "show and follow specified logs")
	logsCommand.Flags().IntVarP(&numLines, "lines", "n", 25, "number of lines to show")
	logsCommand.Flags().StringVarP(&procName, "process", "p", "", "optional process name")
	logsCommand.Flags().BoolVarP(&quiet, "quiet", "q", false, "suppress filename headers")

	RootCmd.AddCommand(logsCommand)
}

var logsCommand = &cobra.Command{
	RunE:    logsForJob,
	Short:   "streams the logs for a given job",
	Use:     "logs <job-name>",
	PreRunE: logsPre,
}

func logsPre(cmd *cobra.Command, args []string) error {
	return validateInput(args)
}

func logsForJob(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	var filesToTail []string
	var tailArgs []string

	if shouldTailStdout() {
		filesToTail = append(filesToTail, bpmCfg.Stdout().External())
	}

	if shouldTailStderr() {
		filesToTail = append(filesToTail, bpmCfg.Stderr().External())
	}

	if logsDontExist(filesToTail) {
		return errors.New("logs not found")
	}

	if follow {
		tailArgs = append(tailArgs, "-f")
	}

	if quiet {
		tailArgs = append(tailArgs, "-q")
	}

	linesToPrint := fmt.Sprintf("-n %d", numLines)
	tailArgs = append(tailArgs, linesToPrint)

	tailArgs = append(tailArgs, filesToTail...)
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

	signals := make(chan os.Signal, 1)
	signal.Notify(signals)

	for {
		select {
		case sig := <-signals: // Forward signal received by parent to child
			tailCmd.Process.Signal(sig)
		case err := <-errCh: // Signal parent when child dies
			if err != nil && err.Error() != "signal: interrupt" {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), "")
			return nil
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
