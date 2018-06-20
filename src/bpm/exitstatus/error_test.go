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

package exitstatus_test

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bpm/exitstatus"
)

func TestExitStatus(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Exit Status Suite")
}

var _ = Describe("exit status", func() {
	Describe("message", func() {
		It("includes the exit status", func() {
			msg := "disaster"
			err := &exitstatus.Error{
				Status: 34,
				Err:    errors.New(msg),
			}

			Expect(err.Error()).To(Equal("disaster (exit status 34)"))
		})
	})

	Describe("getting the exit status", func() {
		Context("with no error", func() {
			It("returns 0", func() {
				status := exitstatus.FromError(nil)
				Expect(status).To(BeZero())
			})
		})

		Context("with an exit status error", func() {
			It("returns the exit status from the error", func() {
				err := &exitstatus.Error{Status: 41, Err: errors.New("oops")}
				status := exitstatus.FromError(err)
				Expect(status).To(Equal(41))
			})
		})

		Context("with an other error", func() {
			It("returns a generic 1", func() {
				err := errors.New("other")
				status := exitstatus.FromError(err)
				Expect(status).To(Equal(1))
			})
		})
	})
})
