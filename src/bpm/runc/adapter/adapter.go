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

package adapter

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"code.cloudfoundry.org/bytefmt"
	"code.cloudfoundry.org/lager"
	specs "github.com/opencontainers/runtime-spec/specs-go"

	"bpm/config"
	"bpm/runc/specbuilder"
	"bpm/sysfeat"
)

const (
	resolvConfDir = "/run/resolvconf"
	defaultLang   = "en_US.UTF-8"

	procSysFsNrOpenFile = "/proc/sys/fs/nr_open"

	// LinuxDefaultMaxOpenFileLimit --  this value comes from fs/file.c in the
	// Linux kernel source.  As of v5 of the kernel, it is set to the value
	// below.
	LinuxDefaultMaxOpenFileLimit = 1024 * 1024
)

// GlobFunc is a function which when given a file path pattern returns a list
// of paths or an error if the search failed.
type GlobFunc func(string) ([]string, error)

type RuncAdapter struct {
	features sysfeat.Features
	glob     GlobFunc
}

func NewRuncAdapter(features sysfeat.Features, glob GlobFunc) *RuncAdapter {
	return &RuncAdapter{
		features: features,
		glob:     glob,
	}
}

func (a *RuncAdapter) CreateJobPrerequisites(
	bpmCfg *config.BPMConfig,
	procCfg *config.ProcessConfig,
	user specs.User,
) (*os.File, *os.File, error) {
	err := os.MkdirAll(bpmCfg.PidDir(), 0700)
	if err != nil {
		return nil, nil, err
	}

	var dirsToCreate, pathsToChown []string
	for _, vol := range procCfg.AdditionalVolumes {
		if vol.MountOnly {
			continue
		}

		_, err = os.Stat(vol.Path)
		if os.IsNotExist(err) {
			dirsToCreate = append(dirsToCreate, vol.Path)
			continue
		}

		if err != nil {
			return nil, nil, err
		}

		pathsToChown = append(pathsToChown, vol.Path)
	}

	dirsToCreate = append(
		dirsToCreate,
		bpmCfg.LogDir(),
		bpmCfg.SocketDir(),
		bpmCfg.TempDir(),
	)

	if procCfg.EphemeralDisk {
		dirsToCreate = append(dirsToCreate, bpmCfg.DataDir())
	}

	if procCfg.PersistentDisk {
		var storeExists bool
		storeExists, err = checkDirExists(filepath.Dir(bpmCfg.StoreDir()))
		if err != nil {
			return nil, nil, err
		}

		if !storeExists {
			return nil, nil, errors.New("requested persistent disk does not exist")
		}

		dirsToCreate = append(dirsToCreate, bpmCfg.StoreDir())
	}

	err = createDirs(dirsToCreate, user)
	if err != nil {
		return nil, nil, err
	}

	err = chownPaths(pathsToChown, user)
	if err != nil {
		return nil, nil, err
	}

	return createLogFiles(bpmCfg, user)
}

func createDirs(dirs []string, user specs.User) error {
	for _, dir := range dirs {
		err := createDirFor(dir, int(user.UID), int(user.GID))
		if err != nil {
			return err
		}
	}

	return nil
}

func createDirFor(path string, uid, gid int) error {
	err := os.MkdirAll(path, 0700)
	if err != nil {
		return err
	}

	return os.Chown(path, uid, gid)
}

func chownPaths(paths []string, user specs.User) error {
	for _, path := range paths {
		err := os.Chown(path, int(user.UID), int(user.GID))
		if err != nil {
			return err
		}
	}

	return nil
}

func createLogFiles(bpmCfg *config.BPMConfig, user specs.User) (*os.File, *os.File, error) {
	files := make([]*os.File, 2)
	paths := []string{bpmCfg.Stdout(), bpmCfg.Stderr()}
	for i, path := range paths {
		f, err := createFileFor(path, int(user.UID), int(user.GID))
		if err != nil {
			return nil, nil, err
		}
		files[i] = f
	}

	return files[0], files[1], nil
}

func createFileFor(path string, uid, gid int) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}

	err = os.Chown(path, uid, gid)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func retrieveMaxOpenFileLimit() (int64, error) {
	b, err := ioutil.ReadFile(procSysFsNrOpenFile)
	if err != nil {
		return LinuxDefaultMaxOpenFileLimit, err
	}

	maxOpenFiles, err := strconv.Atoi(string(bytes.TrimSpace(b)))
	if err != nil {
		return LinuxDefaultMaxOpenFileLimit, err
	}

	return int64(maxOpenFiles), nil
}

func (a *RuncAdapter) BuildSpec(
	logger lager.Logger,
	bpmCfg *config.BPMConfig,
	procCfg *config.ProcessConfig,
	user specs.User,
) (specs.Spec, error) {
	cwd := bpmCfg.JobDir()
	if procCfg.WorkDir != "" {
		cwd = procCfg.WorkDir
	}

	mountResolvConf, err := checkDirExists(resolvConfDir)
	if err != nil {
		return specs.Spec{}, err
	}

	ms := newMountDedup(logger)
	ms.addMounts(systemIdentityMounts(mountResolvConf))
	ms.addMounts(boshMounts(bpmCfg, procCfg.EphemeralDisk, procCfg.PersistentDisk))
	ms.addMounts(userProvidedIdentityMounts(bpmCfg, procCfg.AdditionalVolumes))
	if procCfg.Unsafe != nil && len(procCfg.Unsafe.UnrestrictedVolumes) > 0 {
		expanded, err := a.globExpandVolumes(procCfg.Unsafe.UnrestrictedVolumes)
		if err != nil {
			return specs.Spec{}, err
		}
		ms.addMounts(userProvidedIdentityMounts(bpmCfg, expanded))
	}

	spec := specbuilder.Build(
		specbuilder.WithRootFilesystem(bpmCfg.RootFSPath()),
		specbuilder.WithUser(user),
		specbuilder.WithProcess(
			procCfg.Executable,
			procCfg.Args,
			processEnvironment(procCfg.Env, bpmCfg),
			cwd,
		),
		specbuilder.WithCapabilities(processCapabilities(procCfg.Capabilities)),
		specbuilder.WithMounts(ms.mounts()),
		specbuilder.WithNamespace("ipc"),
		specbuilder.WithNamespace("mount"),
		specbuilder.WithNamespace("pid"),
		specbuilder.WithNamespace("uts"),
	)

	if procCfg.Limits != nil {
		if procCfg.Limits.Memory != nil {
			memLimit, err := bytefmt.ToBytes(*procCfg.Limits.Memory)
			if err != nil {
				return specs.Spec{}, err
			}

			specbuilder.Apply(spec, specbuilder.WithMemoryLimit(int64(memLimit), a.features))
		}

		if procCfg.Limits.Processes != nil {
			specbuilder.Apply(spec, specbuilder.WithPidLimit(*procCfg.Limits.Processes))
		}

		if procCfg.Limits.OpenFiles != nil {
			openFiles := *procCfg.Limits.OpenFiles
			if openFiles == -1 {
				openFiles, err = retrieveMaxOpenFileLimit()
				if err != nil {
					logger.Error("failed-to-retrieve-maximum-open-file-limit", err)
					logger.Info("defaulting-to-maximum-open-file-limit")
				}
			}
			specbuilder.Apply(spec, specbuilder.WithOpenFileLimit(uint64(openFiles)))
		}
	}

	if procCfg.Unsafe != nil && procCfg.Unsafe.Privileged {
		specbuilder.Apply(spec, specbuilder.WithPrivileged())
	}

	return *spec, nil
}

func systemIdentityMounts(mountResolvConf bool) []specs.Mount {
	mounts := []specs.Mount{
		identityBindMountWithOptions("/bin", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/etc", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/lib", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/lib64", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/sbin", "nosuid", "nodev", "bind", "ro"),
		identityBindMountWithOptions("/usr", "nosuid", "nodev", "bind", "ro"),
	}

	if mountResolvConf {
		mounts = append(mounts, identityBindMountWithOptions("/run/resolvconf", "nodev", "nosuid", "noexec", "bind", "ro"))
	}

	return mounts
}

func boshMounts(bpmCfg *config.BPMConfig, mountData, mountStore bool) []specs.Mount {
	mounts := []specs.Mount{
		bindMountWithOptions("/tmp", bpmCfg.TempDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		bindMountWithOptions("/var/tmp", bpmCfg.TempDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		identityBindMountWithOptions(bpmCfg.DataPackageDir(), "nodev", "nosuid", "bind", "ro"),
		identityBindMountWithOptions(bpmCfg.JobDir(), "nodev", "nosuid", "bind", "ro"),
		identityBindMountWithOptions(bpmCfg.LogDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
		identityBindMountWithOptions(bpmCfg.PackageDir(), "nodev", "nosuid", "bind", "ro"),
		identityBindMountWithOptions(bpmCfg.TempDir(), "nodev", "nosuid", "noexec", "rbind", "rw"),
	}

	if mountData {
		mounts = append(mounts, identityBindMountWithOptions(bpmCfg.DataDir(), "nodev", "nosuid", "noexec", "rbind", "rw"))
	}

	if mountStore {
		mounts = append(mounts, identityBindMountWithOptions(bpmCfg.StoreDir(), "nodev", "nosuid", "noexec", "rbind", "rw"))
	}

	return mounts
}

func (a *RuncAdapter) globExpandVolumes(volumes []config.Volume) ([]config.Volume, error) {
	var expandedVolumes []config.Volume

	for _, volume := range volumes {
		matches, err := a.glob(volume.Path)
		if err != nil {
			return nil, err
		}

		for _, match := range matches {
			v := volume
			v.Path = match
			expandedVolumes = append(expandedVolumes, v)
		}
	}

	return expandedVolumes, nil
}

func userProvidedIdentityMounts(bpmCfg *config.BPMConfig, volumes []config.Volume) []specs.Mount {
	var mnts []specs.Mount

	for _, vol := range volumes {
		execOpt := "noexec"
		if vol.AllowExecutions {
			execOpt = "exec"
		}

		writeOpt := "ro"
		if vol.Writable {
			writeOpt = "rw"
		}

		mnts = append(mnts, identityBindMountWithOptions(vol.Path, "nodev", "nosuid", execOpt, "rbind", writeOpt))
	}

	return mnts
}

func identityBindMountWithOptions(path string, options ...string) specs.Mount {
	return bindMountWithOptions(path, path, options...)
}

func bindMountWithOptions(dest, src string, options ...string) specs.Mount {
	return specs.Mount{
		Destination: dest,
		Type:        "bind",
		Source:      src,
		Options:     options,
	}
}

type dedupMounts struct {
	set    map[string]specs.Mount
	logger lager.Logger
}

func newMountDedup(logger lager.Logger) *dedupMounts {
	return &dedupMounts{
		set:    make(map[string]specs.Mount),
		logger: logger,
	}
}

func (d *dedupMounts) addMounts(ms []specs.Mount) {
	for _, mount := range ms {
		dst := mount.Destination
		if _, ok := d.set[dst]; ok {
			d.logger.Info("duplicate-mount", lager.Data{"mount": dst})
			continue
		}
		d.set[dst] = mount
	}
}

func (d *dedupMounts) mounts() []specs.Mount {
	ms := make([]specs.Mount, 0, len(d.set))

	for _, mount := range d.set {
		ms = append(ms, mount)
	}

	sort.Slice(ms, func(i, j int) bool {
		iElems := strings.Split(ms[i].Destination, "/")
		jElems := strings.Split(ms[j].Destination, "/")
		return len(iElems) < len(jElems)
	})

	return ms
}

func processEnvironment(env map[string]string, cfg *config.BPMConfig) []string {
	var environ []string

	for k, v := range env {
		environ = append(environ, fmt.Sprintf("%s=%s", k, v))
	}

	if _, ok := env["TMPDIR"]; !ok {
		environ = append(environ, fmt.Sprintf("TMPDIR=%s", cfg.TempDir()))
	}

	if _, ok := env["LANG"]; !ok {
		environ = append(environ, fmt.Sprintf("LANG=%s", defaultLang))
	}

	if _, ok := env["PATH"]; !ok {
		environ = append(environ, fmt.Sprintf("PATH=%s", defaultPath(cfg)))
	}

	if _, ok := env["HOME"]; !ok {
		environ = append(environ, fmt.Sprintf("HOME=%s", cfg.DataDir()))
	}

	return environ
}

func processCapabilities(caps []string) []string {
	var capsWithPrefix []string

	for _, cap := range caps {
		capsWithPrefix = append(capsWithPrefix, fmt.Sprintf("CAP_%s", cap))
	}

	return capsWithPrefix
}

func checkDirExists(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	return false, nil
}

func defaultPath(cfg *config.BPMConfig) string {
	defaultPath := "%s:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin:."
	defaultPath = fmt.Sprintf(defaultPath, filepath.Join(cfg.JobDir(), "bin"))

	return defaultPath
}
