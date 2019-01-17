// Copyright (C) 2019-Present CloudFoundry.org Foundation, Inc. All rights reserved.
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

package stopsched

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/clock/fakeclock"
)

var _ = Describe("Schedule", func() {
	It("does nothing with an empty schedule", func() {
		ctx := context.Background()
		s, err := Parse("")
		Expect(err).NotTo(HaveOccurred())

		err = Run(ctx, s)
		Expect(err).NotTo(HaveOccurred())
	})

	It("triggers an associated action when reached", func() {
		ctx := context.Background()
		s, err := Parse("FIRST")
		Expect(err).NotTo(HaveOccurred())

		triggered := false

		err = Run(ctx, s, WithActions(Actions{
			"FIRST": func() error {
				triggered = true
				return nil
			},
		}))
		Expect(err).NotTo(HaveOccurred())

		Expect(triggered).To(BeTrue())
	})

	It("triggers multiple associated actions", func() {
		ctx := context.Background()
		s, err := Parse("FIRST/SECOND")
		Expect(err).NotTo(HaveOccurred())

		triggeredFirst := false
		triggeredSecond := false

		err = Run(ctx, s, WithActions(Actions{
			"FIRST": func() error {
				triggeredFirst = true
				return nil
			},
			"SECOND": func() error {
				triggeredSecond = true
				return nil
			},
		}))
		Expect(err).NotTo(HaveOccurred())

		Expect(triggeredFirst).To(BeTrue())
		Expect(triggeredSecond).To(BeTrue())
	})

	It("pauses in between actions", func() {
		ctx := context.Background()
		s, err := Parse("FIRST/10/SECOND")
		Expect(err).NotTo(HaveOccurred())

		start := time.Now()
		c := fakeclock.NewFakeClock(start)

		triggeredFirst := false
		triggeredSecond := false

		err = Run(ctx, s, WithActions(Actions{
			"FIRST": func() error {
				triggeredFirst = true
				go c.WaitForWatcherAndIncrement(10 * time.Second)
				return nil
			},
			"SECOND": func() error {
				Expect(c.Now()).To(BeTemporally("==", start.Add(10*time.Second)))
				triggeredSecond = true
				return nil
			},
		}), WithClock(c))
		Expect(err).NotTo(HaveOccurred())

		Expect(triggeredFirst).To(BeTrue())
		Expect(triggeredSecond).To(BeTrue())
	})

	It("returns an error and exits immediately if a step errors", func() {
		ctx := context.Background()
		s, err := Parse("ERR/1000")
		Expect(err).NotTo(HaveOccurred())

		err = Run(ctx, s, WithActions(Actions{
			"ERR": func() error {
				return errors.New("disaster")
			},
		}))
		Expect(err).To(MatchError("disaster"))
	})

	It("returns an error and exits if it doesn't know how to perform an action", func() {
		ctx := context.Background()
		s, err := Parse("UNKNOWN")
		Expect(err).NotTo(HaveOccurred())

		err = Run(ctx, s)
		Expect(err).To(MatchError(`unknown action "UNKNOWN"`))
	})
})
