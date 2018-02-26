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
	"log"
	"net/http"

	"golang.org/x/sys/unix"
)

func SyscallDisallowed(w http.ResponseWriter, r *http.Request) {
	// The actual device number does not matter as the error should be the same
	// regardless of whether the device exists or not.
	deviceNumber := 1234
	err := unix.Ustat(deviceNumber, &unix.Ustat_t{})
	if err != nil {
		if err.Error() != "operation not permitted" {

			log.Printf("SyscallDisallowed - unexpected error occurred: %s\n", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Unexpected error occurred: %s", err.Error())
			return
		}

		fmt.Printf("SyscallDisallowed - expected error occurred: %s\n", err.Error())

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Expected error occurred: %s", err.Error())
		return
	}

	log.Println("SyscallDisallowed - expected error did not occur")

	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "Expected error did not occur")
}
