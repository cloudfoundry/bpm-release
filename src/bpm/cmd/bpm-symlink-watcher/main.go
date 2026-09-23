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

// Command bpm-symlink-watcher keeps /var/vcap/jobs/bpm/bin/bpm resolving to
// the stemcell's bpm, /usr/bin/bpm.
//
// Every BPM consumer invokes /var/vcap/jobs/bpm/bin/bpm by absolute path. The
// stemcell satisfies that with a symlink. When a deployment provides its own
// bpm release, the BOSH agent replaces /var/vcap/jobs/bpm with a link to its
// own job. When a deployment later removes bpm, the agent deletes that link.
//
// bpm-symlink-watcher runs on boot (ensuring symlinks exist before bosh-agent starts)
// and watches /var/vcap/bosh using inotify for spec.json completion (IN_CLOSE_WRITE).
// When an apply finishes, the watcher immediately checks whether the symlinks are
// missing or dangling, and atomically restores them.
package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	boshDir         = "/var/vcap/bosh"
	heartbeatPeriod = 60 * time.Second
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	logger := log.Default()

	reconciler := NewReconciler(DefaultArtifacts, logger)

	// Step 1: Initial boot-time reconciliation pass
	logger.Println("running initial boot reconciliation")
	if restored, err := reconciler.ReconcileAll(); err != nil {
		logger.Printf("error during initial reconciliation: %v", err)
	} else if len(restored) > 0 {
		logger.Printf("restored artifacts on boot: %v", restored)
	}

	// Step 2: Start directory watcher on /var/vcap/bosh for spec.json
	specEvents := make(chan struct{}, 1)
	watcher, err := startBoshDirWatcher(boshDir, specEvents, logger)
	if err != nil {
		logger.Fatalf("failed to start directory watcher: %v", err)
	}
	defer func() {
		if err := watcher.Close(); err != nil {
			logger.Printf("failed to close directory watcher: %v", err)
		}
	}()

	// Step 3: Low-frequency passive heartbeat as fallback safety net
	heartbeat := time.NewTicker(heartbeatPeriod)
	defer heartbeat.Stop()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Println("bpm-symlink-watcher started and waiting for events")

	for {
		select {
		case <-specEvents:
			logger.Println("spec.json updated; evaluating symlink restoration")
			if restored, err := reconciler.ReconcileAll(); err != nil {
				logger.Printf("error reconciling artifacts: %v", err)
			} else if len(restored) > 0 {
				logger.Printf("restored artifacts after apply: %v", restored)
			}
		case <-heartbeat.C:
			if restored, err := reconciler.ReconcileAll(); err != nil {
				logger.Printf("error during heartbeat reconciliation: %v", err)
			} else if len(restored) > 0 {
				logger.Printf("restored artifacts during heartbeat: %v", restored)
			}
		case sig := <-sigCh:
			logger.Printf("received signal %v; shutting down", sig)
			return
		}
	}
}
