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

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"bpm/acceptance/fixtures/test-server/handlers"
)

var (
	port          = flag.Int("port", -1, "port the server listens on")
	ignoreSignals = flag.Bool("ignore-signals", false, "ignore all signals")
)

func main() {
	flag.Parse()
	if *port == -1 {
		log.Fatal("no explicit port specified")
	}

	bpmVar := os.Getenv("BPM")
	if bpmVar == "" {
		log.Fatal("BPM environment variable not set")
	}

	fmt.Println("Test Agent Started - STDOUT")
	log.Println("Test Agent Started - STDERR")

	http.HandleFunc("/", handlers.Hello)
	http.HandleFunc("/curl", handlers.Curl)
	http.HandleFunc("/mounts", handlers.Mounts)
	http.HandleFunc("/masked-paths", handlers.MaskedPaths)
	http.HandleFunc("/processes", handlers.Processes)
	http.HandleFunc("/syscall-allowed", handlers.SyscallAllowed)
	http.HandleFunc("/syscall-disallowed", handlers.SyscallDisallowed)
	http.HandleFunc("/var-vcap", handlers.VarVcap)
	http.HandleFunc("/var-vcap-data", handlers.VarVcapData)
	http.HandleFunc("/var-vcap-jobs", handlers.VarVcapJobs)
	http.HandleFunc("/whoami", handlers.Whoami)
	http.HandleFunc("/env", handlers.Env)

	signals := make(chan os.Signal)
	signal.Notify(signals)

	go handleSignals(signals)

	addr := fmt.Sprintf(":%d", *port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalln(err)
	}
}

func handleSignals(signals chan os.Signal) {
	for sig := range signals {
		if *ignoreSignals {
			log.Printf("ignoring signal: %#v\n", sig)
			continue
		}

		switch sig {
		case syscall.SIGTERM:
			os.Exit(0)
		case syscall.SIGQUIT:
			// Ignore the error here because if it occurs then there's nothing we can
			// do. We've already failed writing something to standard error!
			_ = pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
			os.Exit(0)
		default:
			log.Printf("unhandled signal: %#v\n", sig)
		}
	}
}
