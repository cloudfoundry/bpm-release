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

package integration_test

import (
	"fmt"
)

func defaultBash(path string) string {
	return fmt.Sprintf(`trap "echo 'Received a Signal' && kill -9 $child" SIGTERM;
echo $LANG;
echo "Logging to STDOUT";
echo "Logging to STDERR" 1>&2;
echo "Logging to FILE" > %s;
sleep 100 &
child=$!;
wait $child`, path)
}

const alternativeBash = `trap "kill -9 $child" SIGTERM;
echo $LANG;
echo "Alternate Logging to STDOUT";
echo "Alternate Logging to STDERR" 1>&2;
sleep 100 &
child=$!;
wait $child`

const preStartBash = `#!/bin/bash
echo "Executing Pre Start"`

const effectiveCapabiltiesBash = `cat /proc/1/status | grep CapEff`

const netBindServiceCapabilityBash = `echo PRIVILEGED | nc -l 127.0.0.1 80`

// See https://codegolf.stackexchange.com/questions/24485/create-a-memory-leak-without-any-fork-bombs
const memoryLeakBash = `start_memory_leak() { :(){ : $@$@;};: : ;};
trap "kill $child" SIGTERM;
sleep 100 &
child=$!;
wait $child;
start_memory_leak`

func fileLeakBash(path string) string {
	return fmt.Sprintf(`file_dir=%s;
start_file_leak() { for i in $(seq 1 20); do touch $file_dir/file-$i; done; tail -f $file_dir/* ;};
trap "kill $child" SIGTERM;
sleep 100 &
child=$!;
wait $child;
start_file_leak`, path)
}

const processLeakBash = `trap "if [ \"$child\" ]; then kill $child; fi" SIGTERM;
sleep 100 &
child=$!;
wait $child;
for i in $(seq 1 999); do sleep 100 & done;
wait`

func messageQueueBash(id int) string {
	return fmt.Sprintf(`ipcs -q -i %d; sleep 5`, id)
}

const logsBash = `trap "kill -9 $child" SIGTERM;
for i in $(seq 1 100); do
  echo "Logging Line #$i to STDOUT"
  echo "Logging Line #$i to STDERR" 1>&2
done

sleep 100 &
child=$!;
wait $child`

const alternativeLogsBash = `trap "kill -9 $child" SIGTERM;
for i in $(seq 1 100); do
  echo "Logging Line #$i to ALT STDOUT"
  echo "Logging Line #$i to ALT STDERR" 1>&2
done

sleep 100 &
child=$!;
wait $child`

const waitForSigUSR1Bash = `trap "kill -9 $child" SIGUSR1;
sleep 100 &
child=$!;
wait $child`
