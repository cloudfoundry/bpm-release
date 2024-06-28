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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/bytefmt"
	"code.cloudfoundry.org/lager/v3"
	specs "github.com/opencontainers/runtime-spec/specs-go"

	"bpm/config"
	"bpm/hostlock"
	"bpm/runc/specbuilder"
	"bpm/sysfeat"
)

const (
	resolvConfDir = "/run/resolvconf"
	defaultLang   = "en_US.UTF-8"
)

// GlobFunc is a function which when given a file path pattern returns a list
// of paths or an error if the search failed.
type GlobFunc func(string) ([]string, error)

type MountShare func(string) error

type VolumeLocker interface {
	LockVolume(string) (hostlock.LockedLock, error)
}

type RuncAdapter struct {
	features   sysfeat.Features
	glob       GlobFunc
	shareMount MountShare
	locker     VolumeLocker
}

func NewRuncAdapter(features sysfeat.Features, glob GlobFunc, mountSharer MountShare, locker VolumeLocker) *RuncAdapter {
	return &RuncAdapter{
		features:   features,
		glob:       glob,
		shareMount: mountSharer,
		locker:     locker,
	}
}

func (a *RuncAdapter) CreateJobPrerequisites(
	bpmCfg *config.BPMConfig,
	procCfg *config.ProcessConfig,
	user specs.User,
) (*os.File, *os.File, error) {
	err := os.MkdirAll(bpmCfg.PidDir().External(), 0700)
	if err != nil {
		return nil, nil, err
	}

	var dirsToCreate, pathsToChown []string
	for _, vol := range procCfg.AdditionalVolumes {
		if vol.Shared {
			if err := a.makeShared(vol); err != nil {
				return nil, nil, err
			}
		}

		if vol.MountOnly {
			continue
		}

		fi, err := os.Stat(vol.Path)
		if os.IsNotExist(err) {
			dirsToCreate = append(dirsToCreate, vol.Path)
		} else if err != nil {
			return nil, nil, err
		} else if fi.IsDir() && fi.Mode() != 0700 {
			if err := os.Chmod(vol.Path, 0700); err != nil {
				return nil, nil, err
			}
		}

		pathsToChown = append(pathsToChown, vol.Path)
	}

	dirsToCreate = append(
		dirsToCreate,
		bpmCfg.LogDir().External(),
		bpmCfg.SocketDir().External(),
		bpmCfg.TempDir().External(),
	)

	if procCfg.EphemeralDisk {
		dirsToCreate = append(dirsToCreate, bpmCfg.DataDir().External())
	}

	if procCfg.PersistentDisk {
		storeDir := bpmCfg.StoreDir().External()
		storeExists, err := checkDirExists(filepath.Dir(storeDir))
		if err != nil {
			return nil, nil, err
		}

		if !storeExists {
			return nil, nil, errors.New("requested persistent disk does not exist")
		}

		dirsToCreate = append(dirsToCreate, storeDir)
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

func (a *RuncAdapter) makeShared(volume config.Volume) error {
	held, err := a.locker.LockVolume(volume.Path)
	if err != nil {
		return err
	}
	defer held.Unlock() //nolint:errcheck

	if err := a.shareMount(volume.Path); err != nil {
		return err
	}

	return nil
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
	paths := []string{bpmCfg.Stdout().External(), bpmCfg.Stderr().External()}
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

func (a *RuncAdapter) BuildSpec(
	logger lager.Logger,
	bpmCfg *config.BPMConfig,
	procCfg *config.ProcessConfig,
	user specs.User,
) (specs.Spec, error) {
	cwd := bpmCfg.JobDir().Internal()
	if procCfg.WorkDir != "" {
		cwd = procCfg.WorkDir
	}

	mountResolvConf, err := checkDirExists(resolvConfDir)
	if err != nil {
		return specs.Spec{}, err
	}

	ms := newMountDedup(logger)
	ms.addMounts(systemIdentityMounts(mountResolvConf))
	boshMounts := boshMounts(bpmCfg, procCfg.EphemeralDisk, procCfg.PersistentDisk)
	ms.addMounts(boshMounts)
	ms.addMounts(userProvidedIdentityMounts(bpmCfg, procCfg.AdditionalVolumes))
	if procCfg.Unsafe != nil && len(procCfg.Unsafe.UnrestrictedVolumes) > 0 {
		expanded, err := a.globExpandVolumes(procCfg.Unsafe.UnrestrictedVolumes)
		if err != nil {
			return specs.Spec{}, err
		}
		filteredVolumes := filterVolumesUnderBoshMounts(boshMounts, expanded)
		ms.addMounts(userProvidedIdentityMounts(bpmCfg, filteredVolumes))
	}

	wrappedExe, wrappedArgs := wrapWithInit(bpmCfg, procCfg)

	spec := specbuilder.Build(
		specbuilder.WithRootFilesystem(bpmCfg.RootFSPath()),
		specbuilder.WithUser(user),
		specbuilder.WithProcess(
			wrappedExe,
			wrappedArgs,
			processEnvironment(procCfg.Env, bpmCfg),
			cwd,
		),
		specbuilder.WithCapabilities(processCapabilities(procCfg.Capabilities)),
		specbuilder.WithMounts(ms.mounts()),
		specbuilder.WithNamespace("ipc"),
		specbuilder.WithNamespace("mount"),
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
			specbuilder.Apply(spec, specbuilder.WithOpenFileLimit(*procCfg.Limits.OpenFiles))
		}
	}

	if procCfg.Unsafe == nil || !procCfg.Unsafe.HostPidNamespace {
		specbuilder.Apply(spec, specbuilder.WithNamespace("pid"))
	}

	if procCfg.Unsafe != nil && procCfg.Unsafe.Privileged {
		specbuilder.Apply(spec, specbuilder.WithPrivileged())
	}

	return *spec, nil
}

func filterVolumesUnderBoshMounts(mounts []specs.Mount, volumes []config.Volume) []config.Volume {
	var filteredVolumes []config.Volume
	for _, v := range volumes {
		keep := true
		for _, m := range mounts {
			if strings.HasPrefix(v.Path, m.Destination) {
				keep = false
			}
		}

		if keep {
			filteredVolumes = append(filteredVolumes, v)
		}
	}

	return filteredVolumes
}

func wrapWithInit(bpmCfg *config.BPMConfig, procCfg *config.ProcessConfig) (string, []string) {
	exe := bpmCfg.TiniPath().Internal()
	args := append([]string{"-w", "-s", "--", procCfg.Executable}, procCfg.Args...)
	return exe, args
}

func systemIdentityMounts(mountResolvConf bool) []specs.Mount {
	mounts := []specs.Mount{
		IdentityMount("/bin", AllowExec()),
		IdentityMount("/etc", AllowExec()),
		IdentityMount("/lib", AllowExec()),
		IdentityMount("/lib64", AllowExec()),
		IdentityMount("/sbin", AllowExec()),
		IdentityMount("/usr", AllowExec()),
	}

	if mountResolvConf {
		mounts = append(mounts, IdentityMount("/run/resolvconf"))
	}

	return mounts
}

func boshMounts(bpmCfg *config.BPMConfig, mountData, mountStore bool) []specs.Mount {
	jobDir := bpmCfg.JobDir()
	logDir := bpmCfg.LogDir()
	tmpDir := bpmCfg.TempDir()
	packageDir := bpmCfg.PackageDir()
	dataPackageDir := bpmCfg.DataPackageDir()

	mounts := []specs.Mount{
		Mount(tmpDir.External(), "/tmp", WithRecursiveBind(), AllowWrites()),
		Mount(tmpDir.External(), "/var/tmp", WithRecursiveBind(), AllowWrites()),
		Mount(tmpDir.External(), tmpDir.Internal(), WithRecursiveBind(), AllowWrites()),
		Mount(dataPackageDir.External(), dataPackageDir.Internal(), AllowExec()),
		Mount(packageDir.External(), packageDir.Internal(), AllowExec()),
		Mount(jobDir.External(), jobDir.Internal(), AllowExec()),
		Mount(logDir.External(), logDir.Internal(), WithRecursiveBind(), AllowWrites()),
	}

	if mountData {
		dataDir := bpmCfg.DataDir()
		mounts = append(mounts, Mount(dataDir.External(), dataDir.Internal(), WithRecursiveBind(), AllowWrites()))
	}

	if mountStore {
		storeDir := bpmCfg.StoreDir()
		mounts = append(mounts, Mount(storeDir.External(), storeDir.Internal(), WithRecursiveBind(), AllowWrites()))
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
	var mounts []specs.Mount

	for _, vol := range volumes {
		opts := []MountOption{WithRecursiveBind()}

		if vol.AllowExecutions {
			opts = append(opts, AllowExec())
		}

		if vol.Writable {
			opts = append(opts, AllowWrites())
		}

		mounts = append(mounts, IdentityMount(vol.Path, opts...))
	}

	return mounts
}

func processEnvironment(env map[string]string, cfg *config.BPMConfig) []string {
	var environ []string

	for k, v := range env {
		environ = append(environ, fmt.Sprintf("%s=%s", k, v))
	}

	if _, ok := env["TMPDIR"]; !ok {
		environ = append(environ, fmt.Sprintf("TMPDIR=%s", cfg.TempDir().Internal()))
	}

	if _, ok := env["LANG"]; !ok {
		environ = append(environ, fmt.Sprintf("LANG=%s", defaultLang))
	}

	if _, ok := env["PATH"]; !ok {
		environ = append(environ, fmt.Sprintf("PATH=%s", defaultPath(cfg)))
	}

	if _, ok := env["HOME"]; !ok {
		environ = append(environ, fmt.Sprintf("HOME=%s", cfg.DataDir().Internal()))
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
	defaultPathTmpl := "%s:/usr/local/bin:/usr/local/sbin:/usr/bin:/usr/sbin:/bin:/sbin:."
	return fmt.Sprintf(defaultPathTmpl, cfg.JobDir().Join("bin").Internal())
}
