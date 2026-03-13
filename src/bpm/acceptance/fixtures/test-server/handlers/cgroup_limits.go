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

	// Build the runc scope name that bpm uses.
	// Main process: runc-bpm-<job>.scope
	// Named process: runc-bpm-<job>.2e<process>.scope
	var scopeName string
	if process == "" || process == job {
		scopeName = fmt.Sprintf("runc-bpm-%s.scope", job)
	} else {
		scopeName = fmt.Sprintf("runc-bpm-%s.2e%s.scope", job, process)
	}

	cgroupDir, err := findCgroupDir("/sys/fs/cgroup", scopeName)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cgroupLimitsResponse{ //nolint:errcheck
			Error: fmt.Sprintf("could not find cgroup dir for scope %s: %v", scopeName, err),
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

// findCgroupDir walks the cgroup filesystem looking for a directory with the
// given scope name. Returns the full path to the scope directory.
func findCgroupDir(root, scopeName string) (string, error) {
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && d.Name() == scopeName {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("scope %s not found under %s", scopeName, root)
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
