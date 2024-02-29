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

package handlers

import (
	"fmt"
	"net/http"
	"os"
)

func VarVcap(w http.ResponseWriter, r *http.Request) {
	items, err := os.ReadDir("/var/vcap")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		fmt.Fprintln(w, item.Name())
	}
}

func VarVcapJobs(w http.ResponseWriter, r *http.Request) {
	items, err := os.ReadDir("/var/vcap/jobs")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		fmt.Fprintln(w, item.Name())
	}
}

func VarVcapData(w http.ResponseWriter, r *http.Request) {
	items, err := os.ReadDir("/var/vcap/data")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	for _, item := range items {
		fmt.Fprintln(w, item.Name())
	}
}
