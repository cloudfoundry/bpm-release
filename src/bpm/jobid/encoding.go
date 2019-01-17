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

package jobid

import (
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	symbols      = "_-" // . is not included because we use it for escaping
	combined     = alphanumeric + symbols
)

// Encode encodes an arbitrary container name into a format which is a valid
// container name for runc. The encoding must not be depended upon and can
// change unexpectedly between versions of BPM.
func Encode(name string) string {
	var id strings.Builder
	id.WriteString("bpm-")

	for i := 0; i < len(name); i++ {
		chr := name[i]
		if needsEscaping(chr) {
			id.WriteString(fmt.Sprintf(".%.2x", chr))
		} else {
			id.WriteByte(chr)
		}
	}

	return id.String()
}

// Decode decodes the runc compatible names returned by Encode back into the
// original string. The encoding must not be depended upon and can change
// unexpectedly between versions of BPM.
func Decode(id string) (string, error) {
	if strings.HasPrefix(id, "bpm-") {
		id = id[4:]
	} else {
		return "", fmt.Errorf("invalid job ID (missing prefix): %q", id)
	}

	var name strings.Builder
	for i := 0; i < len(id); i++ {
		chr := id[i]
		if chr == '.' {
			if i+2 > len(id) {
				return "", fmt.Errorf("invalid job ID (incomplete escape sequence): %q", id)
			}

			code := id[i+1 : i+3]
			if res, err := hex.DecodeString(code); err == nil {
				name.Write(res)
			}
			i += 2
		} else {
			name.WriteByte(chr)
		}
	}

	return name.String(), nil
}

// needsEscaping checks whether a character needs to be escaped for a job ID.
// The valid format is `^[\w-\.]+$`.
func needsEscaping(b byte) bool {
	// If it isn't an alphanumeric character or a ., -, or a _, then it always
	// needs to be escaped.
	return !isAlphaNumSym(b)
}

func isAlphaNumSym(b byte) bool {
	return strings.IndexByte(combined, b) != -1
}
