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
	"os"

	"github.com/spf13/cobra"
)

var Version string

func init() {
	RootCmd.AddCommand(versionCommand)
}

var versionCommand = &cobra.Command{
	Long:  "Prints the BPM version",
	Run:   version,
	Short: "Prints the BPM version",
	Use:   "version",
}

func version(cmd *cobra.Command, args []string) {
	if len(args) != 0 {
		cmd.Usage()
		os.Exit(1)
	}

	if Version == "" {
		Version = "[DEV BUILD]"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Version)
}
