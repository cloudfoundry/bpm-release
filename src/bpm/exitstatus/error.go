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

// Package exitstatus allows an exit status to be pushed through an error
// shaped hole. It should not be used to propogate an exit status through a
// deep callstack as a single error wrapping will cause this to break.
package exitstatus

import "fmt"

// Error represents an error and an associated exit status to propogate.
type Error struct {
	Status int
	Err    error
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s (exit status %d)", e.Err, e.Status)
}

// FromError collects the exit status from the passed error if it exists. If it
// finds an error without status code information then it returns 1 for
// backwards compatibility.
func FromError(err error) int {
	if err == nil {
		return 0
	}
	switch serr := err.(type) {
	case *Error:
		return serr.Status
	default:
		return 1
	}
}
