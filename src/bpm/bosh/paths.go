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
package bosh

import "path/filepath"

// DefaultRoot is the standard root directory for all bosh jobs. All ephemeral
// data, persistent data, logs, and configuration etc. are expected to be in a
// directory somewhere inside this one.
const DefaultRoot = "/var/vcap"

// Path represents a filesystem path to some kind of directory inside a BOSH
// root. A Path has an internal and external representation. The external
// representation is the path outside the running BPM job. The internal
// representation is the path when inside the BPM job mount namespace. In
// production these will always be the DefaultRoot.
//
// This abstraction only exists because we want to have multiple roots on a
// single machine so we can run integration tests concurrently. Forcing the
// consumer to chose between the Internal() or External() representation
// removes a category of bugs around external test harness paths being present
// in the test jobs.
type Path struct {
	root string
	dir  string
}

// Internal returns the internal representation of the Path.
func (p Path) Internal() string {
	return filepath.Join(DefaultRoot, p.dir)
}

// Internal returns the external representation of the Path.
func (p Path) External() string {
	return filepath.Join(p.root, p.dir)
}

// Join allows one or more elements to be joined onto a Path (similar to
// filepath.Join). It returns a new Path with the new elements appended.
func (p Path) Join(elements ...string) Path {
	dir := filepath.Join(append([]string{p.dir}, elements...)...)
	return Path{root: p.root, dir: dir}
}

// String implements the Stringer interface but should never be used. This was
// added to avoid the common mistake of doing:
//
//  fmt.{S,F,}Printf("... %s ...", bosh.Path{...})
//
// Which will does not make it clear if the internal or external representation
// is wanted and is likely a bug. Omitting this function returns the standard
// Go representation of the struct which is almost certainly wrong.
func (p Path) String() string {
	panic("bosh.Path String() called! This should never be used: use Internal() or External() instead.")
}
