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
	"bytes"
	"fmt"
	"io/ioutil"
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

// IsMountpoint returns whether or not the given path is a mount point on the
// system.
func IsMountpoint(path string) (bool, error) {
	ms, err := Mounts()
	if err != nil {
		return false, err
	}

	return isMountpoint(ms, path), nil
}

func isMountpoint(ms []Mnt, path string) bool {
	for _, m := range ms {
		if m.MountPoint == path {
			return true
		}
	}
	return false
}

func Mounts() ([]Mnt, error) {
	bs, err := ioutil.ReadFile("/proc/mounts")
	if err != nil {
		return nil, err
	}
	return ParseFstab(bs)
}

// ParseFstab parses byte slices which contain the contents of files formatted
// as described by fstab(5).
func ParseFstab(contents []byte) ([]Mnt, error) {
	var mnts []Mnt

	r := bytes.NewBuffer(contents)
	scanner := bufio.NewScanner(r)
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

// MakeShared takes a path and turns it into a shared mountpoint. Due to the
// requirement of only being able to share an existing mountpoint this function
// will also bind mount a path to itself if it is not already a mountpoint. In
// the case of an error in making the mountpoint shared this identity mount
// will be rolled back.
func MakeShared(path string) error {
	isMount, err := IsMountpoint(path)
	if err != nil {
		return err
	}

	if !isMount {
		if err := Mount(path, path, "", unix.MS_BIND, ""); err != nil {
			return err
		}
	}

	if err := Mount("none", path, "", unix.MS_SHARED, ""); err != nil {
		if !isMount {
			_ = Unmount(path, 0)
			return err
		}
	}

	return nil
}
