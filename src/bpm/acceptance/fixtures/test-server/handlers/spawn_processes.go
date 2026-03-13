// Copyright (C) 2017-Present CloudFoundry.org Foundation, Inc. All rights reserved.
//
// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License");
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

package handlers

import (
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
)

// SpawnProcesses starts N long-lived child processes (sleep 3600) within
// the same pid cgroup. When the pids cgroup limit is exceeded, fork will
// fail and the cgroup's OOM-equivalent mechanism may kill the process.
func SpawnProcesses(w http.ResponseWriter, r *http.Request) {
	countStr := r.URL.Query().Get("count")
	count, err := strconv.Atoi(countStr)
	if err != nil || count <= 0 {
		http.Error(w, "provide ?count=N where N > 0", http.StatusBadRequest)
		return
	}

	spawned := 0
	for i := 0; i < count; i++ {
		cmd := exec.Command("sleep", "3600")
		if err := cmd.Start(); err != nil {
			fmt.Fprintf(w, "spawned %d of %d, fork failed: %v\n", spawned, count, err) //nolint:errcheck
			return
		}
		spawned++
	}

	fmt.Fprintf(w, "spawned %d processes\n", spawned) //nolint:errcheck
}
