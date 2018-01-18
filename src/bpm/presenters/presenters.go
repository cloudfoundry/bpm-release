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

package presenters

import (
	"bpm/config"
	"bpm/models"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
)

func PrintJobs(processes []*models.Process, stdout io.Writer) error {
	tw := tabwriter.NewWriter(stdout, 0, 0, 1, ' ', 0)

	printRow(tw, "Name", "Pid", "Status")
	for _, process := range processes {
		name, err := config.Decode(process.Name)
		if err != nil {
			return err
		}

		pid := "-"
		if process.Pid > 0 {
			pid = strconv.Itoa(process.Pid)
		}

		printRow(tw, name, pid, process.Status)
	}

	return tw.Flush()
}

func printRow(w io.Writer, args ...string) {
	row := strings.Join(args, "\t")
	fmt.Fprintf(w, "%s\n", row)
}
