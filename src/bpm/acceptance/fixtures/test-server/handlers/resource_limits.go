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
	"syscall"
)

type resourceLimitsResponse struct {
	OpenFiles string `json:"open_files"`
}

func ResourceLimits(w http.ResponseWriter, r *http.Request) {
	resp := resourceLimitsResponse{
		OpenFiles: readOpenFilesLimit(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func readOpenFilesLimit() string {
	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return fmt.Sprintf("%d", rlimit.Max)
}
