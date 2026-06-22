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
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"bpm/jobid"
)

type cgroupLimitsResponse struct {
	MemoryMax string `json:"memory_max"`
	PidsMax   string `json:"pids_max"`
	CgroupDir string `json:"cgroup_dir,omitempty"`
	Error     string `json:"error,omitempty"`
}

// CgroupLimits reads cgroup limit files for a given bpm process by searching
// the host's /sys/fs/cgroup filesystem. This handler is intended to run in a
// privileged bpm container that has /sys/fs/cgroup mounted.
//
// Query params:
//   - process: the bpm process name (e.g. "test-server", "alt-test-server")
//   - job: the bpm job name (default: "test-server")
func CgroupLimits(w http.ResponseWriter, r *http.Request) {
	process := r.URL.Query().Get("process")
	job := r.URL.Query().Get("job")
	if job == "" {
		job = "test-server"
	}

	// Build the container ID exactly the way bpm does (config.BPMConfig.ContainerID):
	// the bare job name for the main process, or "<job>.<process>" for a named
	// process, run through jobid.Encode. Reusing jobid.Encode keeps this in sync
	// if the encoding ever changes.
	name := job
	if process != "" && process != job {
		name = fmt.Sprintf("%s.%s", job, process)
	}
	containerID := jobid.Encode(name)

	cgroupDir, err := findCgroupDir("/sys/fs/cgroup", containerID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cgroupLimitsResponse{ //nolint:errcheck
			Error: fmt.Sprintf("could not find cgroup dir for container %s: %v", containerID, err),
		})
		return
	}

	resp := cgroupLimitsResponse{
		CgroupDir: cgroupDir,
		MemoryMax: readCgroupValue(cgroupDir, "memory.max"),
		PidsMax:   readCgroupValue(cgroupDir, "pids.max"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// findCgroupDir walks the cgroup filesystem looking for the cgroup directory
// for the given bpm container ID. It handles two naming conventions:
//
//   - cgroupfs mode (cgroup v2 without systemd): a directory named exactly
//     containerID, e.g. "bpm-test-server"
//   - systemd mode: a scope directory whose name ends with "-<containerID>.scope",
//     e.g. "runc-bpm-test-server.scope" (legacy) or
//     "garden-abc-scope-bpm-bpm-test-server.scope" (cgroup-v2-aware naming)
//
// Returns the full path to the matching directory.
func findCgroupDir(root, containerID string) (string, error) {
	scopeSuffix := "-" + containerID + ".scope"
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == containerID || strings.HasSuffix(name, scopeSuffix) {
				found = path
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("no cgroup dir found for container %s under %s", containerID, root)
	}
	return found, nil
}

func readCgroupValue(dir, filename string) string {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return strings.TrimSpace(string(data))
}
