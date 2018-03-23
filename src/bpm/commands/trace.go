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

	"bpm/models"
	"bpm/runc/lifecycle"
)

var longText = `
traces a BOSH Process

  Executes strace with the following options:
    strace -s 100 -f -y -yy -p <process-pid>

  <process-pid> is determined using the 'bpm pid' command

  Note: This command may impact performance.
`

func init() {
	traceCommand.Flags().StringVarP(&procName, "process", "p", "", "The optional process name.")
	RootCmd.AddCommand(traceCommand)
}

var traceCommand = &cobra.Command{
	Long:    longText,
	RunE:    trace,
	Short:   "traces a BOSH Process",
	Use:     "trace <job-name>",
	PreRunE: tracePre,
}

func tracePre(cmd *cobra.Command, args []string) error {
	return validateInput(args)
}

func trace(cmd *cobra.Command, _ []string) error {
	cmd.SilenceUsage = true

	runcLifecycle := newRuncLifecycle()

	process, err := runcLifecycle.StatProcess(bpmCfg)
	if lifecycle.IsNotExist(err) || process.Status == models.ProcessStateFailed {
		return errors.New("process is not running or could not be found")
	} else if err != nil {
		return fmt.Errorf("failed to get process: %s", err)
	}

	straceCmd := exec.Command("strace", "-s", "100", "-f", "-y", "-yy", "-p", fmt.Sprintf("%d", process.Pid))
	straceCmd.Stdin = os.Stdin
	straceCmd.Stdout = cmd.OutOrStdout()
	straceCmd.Stderr = cmd.OutOrStderr()

	err = straceCmd.Start()
	if err != nil {
		return err
	}

	cmd.SilenceUsage = true

	errCh := make(chan error)
	go func() {
		errCh <- straceCmd.Wait()
	}()

	signals := make(chan os.Signal)
	signal.Notify(signals)

	for {
		select {
		case sig := <-signals:
			straceCmd.Process.Signal(sig)
		case err := <-errCh:
			return err
		}
	}
}
