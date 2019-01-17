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
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.cloudfoundry.org/clock"
)

func Parse(schedule string) (Schedule, error) {
	var steps []step

	parts := strings.Split(schedule, "/")
	for _, part := range parts {
		if part == "" {
			continue
		}

		if n, err := strconv.Atoi(part); err == nil {
			steps = append(steps, pauseStep{duration: time.Duration(n) * time.Second})
		} else {
			steps = append(steps, actionStep{key: part})
		}
	}

	return Schedule{
		c:       clock.NewClock(),
		steps:   steps,
		actions: make(map[string]func() error),
	}, nil
}

func Run(ctx context.Context, s Schedule, opts ...RunOption) error {
	for _, opt := range opts {
		opt(&s)
	}

	var runErr error
	for _, step := range s.steps {
		if err := step.run(ctx, s); err != nil {
			runErr = err
			break
		}
	}

	if s.ensure != nil {
		if err := s.ensure(); err != nil {
			return err
		}
	}

	return runErr
}

type RunOption func(*Schedule)

type Schedule struct {
	c clock.Clock

	steps   []step
	actions map[string]func() error
	ensure  func() error
}

type step interface {
	run(ctx context.Context, s Schedule) error
}

type pauseStep struct {
	duration time.Duration
}

func (p pauseStep) run(ctx context.Context, s Schedule) error {
	select {
	case <-ctx.Done():
	case <-s.c.After(p.duration):
	}

	return nil
}

type actionStep struct {
	key string
}

func (a actionStep) run(ctx context.Context, s Schedule) error {
	if fn, found := s.actions[a.key]; found {
		return fn()
	}

	return fmt.Errorf("unknown action %q", a.key)
}

func WithClock(c clock.Clock) RunOption {
	return func(s *Schedule) {
		s.c = c
	}
}

func WithAction(key string, fn func() error) RunOption {
	return func(s *Schedule) {
		s.actions[key] = fn
	}
}

func WithEnsure(fn func() error) RunOption {
	return func(s *Schedule) {
		s.ensure = fn
	}
}

type Actions map[string]func() error

func WithActions(as Actions) RunOption {
	return func(s *Schedule) {
		for key, fn := range as {
			s.actions[key] = fn
		}
	}
}
