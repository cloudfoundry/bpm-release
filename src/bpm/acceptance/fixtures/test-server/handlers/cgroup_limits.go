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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
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

type selfCgroupPathResponse struct {
	CgroupV2Path string `json:"cgroup_v2_path,omitempty"`
	Error        string `json:"error,omitempty"`
}

// SelfCgroupPath returns the cgroup v2 unified-mode path of the calling
// process by reading /proc/self/cgroup. The returned path is relative to the
// cgroup root (e.g. "/docker/CONTAINERID/system.slice/monit-service-bpm-bpm-test-server.scope").
// Callers pass this to the privileged observer's CgroupLimits as the cgroup-path
// parameter to read limits from the exact live cgroup.
func SelfCgroupPath(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	f, err := os.Open("/proc/self/cgroup")
	if err != nil {
		if encErr := json.NewEncoder(w).Encode(selfCgroupPathResponse{
			Error: fmt.Sprintf("open /proc/self/cgroup: %v", err),
		}); encErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	defer f.Close() //nolint:errcheck

	path, err := parseCgroupV2Path(f)
	if err != nil {
		if encErr := json.NewEncoder(w).Encode(selfCgroupPathResponse{Error: err.Error()}); encErr != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if err := json.NewEncoder(w).Encode(selfCgroupPathResponse{CgroupV2Path: path}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// parseCgroupV2Path reads a /proc/<pid>/cgroup-format stream and returns the
// cgroup v2 unified-mode path — the entry whose hierarchy ID is 0 (format: "0::<path>").
func parseCgroupV2Path(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "0::") {
			return strings.TrimPrefix(line, "0::"), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading cgroup file: %w", err)
	}
	return "", fmt.Errorf("no cgroup v2 unified-mode entry (0::<path>) in /proc/self/cgroup")
}

// CgroupLimits reads cgroup limit files from an exact path previously obtained
// from the target process via SelfCgroupPath. This handler is intended to run
// in a privileged bpm container that has /sys/fs/cgroup mounted.
//
// Query params:
//   - cgroup-path: the cgroup v2 path returned by SelfCgroupPath on the target
//     process (e.g. "/docker/CONTAINERID/system.slice/monit-service-bpm-bpm-test-server.scope").
//     Reading from the live process's own path avoids stale-cgroup and
//     concurrent-build interference entirely.
func CgroupLimits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	exactPath := r.URL.Query().Get("cgroup-path")
	if exactPath == "" {
		if err := json.NewEncoder(w).Encode(cgroupLimitsResponse{
			Error: "cgroup-path query parameter is required",
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	// Canonicalize before use so that sequences like "/../etc" in exactPath
	// cannot escape the cgroup root.
	const cgroupRoot = "/sys/fs/cgroup"
	cgroupDir := filepath.Clean(cgroupRoot + exactPath)
	if cgroupDir != cgroupRoot && !strings.HasPrefix(cgroupDir, cgroupRoot+"/") {
		if err := json.NewEncoder(w).Encode(cgroupLimitsResponse{
			Error: fmt.Sprintf("cgroup-path %q escapes the cgroup root", exactPath),
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	if err := json.NewEncoder(w).Encode(cgroupLimitsResponse{
		CgroupDir: cgroupDir,
		MemoryMax: readCgroupValue(cgroupDir, "memory.max"),
		PidsMax:   readCgroupValue(cgroupDir, "pids.max"),
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func readCgroupValue(dir, filename string) string {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return strings.TrimSpace(string(data))
}
