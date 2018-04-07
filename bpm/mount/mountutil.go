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

package mount

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

type Mnt struct {
	Device     string
	MountPoint string
	Filesystem string
	Options    []string
}

func Mount(source string, target string, fstype string, flags uintptr, data string) error {
	return unix.Mount(source, target, fstype, flags, data)
}

func Unmount(target string, flags int) error {
	return unix.Unmount(target, flags)
}

func Mounts() ([]Mnt, error) {
	return parseMountFile("/proc/mounts")
}

func parseMountFile(procMountpath string) ([]Mnt, error) {
	f, err := os.Open(procMountpath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mnts := []Mnt{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			return nil, fmt.Errorf("invalid mount: %s", scanner.Text())
		}

		options := strings.Split(fields[3], ",")

		mnts = append(mnts, Mnt{
			Device:     fields[0],
			MountPoint: fields[1],
			Filesystem: fields[2],
			Options:    options,
		})
	}

	return mnts, nil
}
