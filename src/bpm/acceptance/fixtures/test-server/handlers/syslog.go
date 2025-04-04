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

package handlers

import (
	"fmt"
	"log/syslog"
	"net/http"
)

func Syslog(w http.ResponseWriter, r *http.Request) {
	log, err := syslog.New(syslog.LOG_LOCAL0, "")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err) //nolint:errcheck
		return
	}

	if _, err := log.Write([]byte("hello")); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, err) //nolint:errcheck
		return
	}

	fmt.Fprintln(w, "logged!") //nolint:errcheck
}
