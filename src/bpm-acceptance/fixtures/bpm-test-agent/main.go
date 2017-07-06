// Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.

// This program and the accompanying materials are made available under
// the terms of the under the Apache License, Version 2.0 (the "License”);
// you may not use this file except in compliance with the License.

// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"bpm-acceptance/fixtures/bpm-test-agent/handlers"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
)

var port = flag.Int("port", -1, "port the server listens on")

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
	http.HandleFunc("/hostname", handlers.Hostname)
	http.HandleFunc("/mounts", handlers.Mounts)
	http.HandleFunc("/processes", handlers.Processes)
	http.HandleFunc("/var-vcap", handlers.VarVcap)
	http.HandleFunc("/var-vcap-data", handlers.VarVcapData)
	http.HandleFunc("/var-vcap-jobs", handlers.VarVcapJobs)
	http.HandleFunc("/whoami", handlers.Whoami)

	errChan := make(chan error)
	signals := make(chan os.Signal)

	signal.Notify(signals)

	go func() {
		errChan <- http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			log.Fatal(err)
		}
	case sig := <-signals:
		log.Fatalf("Signalled: %#v", sig)
	}
}
