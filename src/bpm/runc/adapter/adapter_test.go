// Copyright (C) 2017-Present Pivotal Software, Inc. All rights reserved.
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

package adapter_test

import (
	"bpm/config"
	"bpm/runc/adapter"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"code.cloudfoundry.org/bytefmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("RuncAdapter", func() {
	var (
		runcAdapter *adapter.RuncAdapter

		jobName,
		procName,
		systemRoot string
		user specs.User

		bpmCfg  *config.BPMConfig
		procCfg *config.ProcessConfig
	)

	BeforeEach(func() {
		runcAdapter = adapter.NewRuncAdapter()

		jobName = "example"
		procName = "server"
		user = specs.User{UID: 200, GID: 300, Username: "vcap"}

		var err error
		systemRoot, err = ioutil.TempDir("", "runc-adapter-system-files")
		Expect(err).NotTo(HaveOccurred())

		bpmCfg = config.NewBPMConfig(systemRoot, jobName, procName)
		procCfg = &config.ProcessConfig{
			Volumes: []string{
				filepath.Join(systemRoot, "some", "directory"),
				filepath.Join(systemRoot, "another", "location"),
			},
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(systemRoot)).To(Succeed())
	})

	Describe("CreateJobPrerequisites", func() {
		It("creates the job prerequisites", func() {
			stdout, stderr, err := runcAdapter.CreateJobPrerequisites(bpmCfg, procCfg, user)
			Expect(err).NotTo(HaveOccurred())

			// PID Directory
			pidDirInfo, err := os.Stat(bpmCfg.PidDir())
			Expect(err).NotTo(HaveOccurred())
			Expect(pidDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(pidDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
			Expect(pidDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(0)))

			// Log Directory
			logDirInfo, err := os.Stat(bpmCfg.LogDir())
			Expect(err).NotTo(HaveOccurred())
			Expect(logDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(logDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(logDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			// Stdout Log File
			Expect(stdout.Name()).To(Equal(bpmCfg.Stdout()))
			stdoutInfo, err := stdout.Stat()
			Expect(err).NotTo(HaveOccurred())
			Expect(stdoutInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0600)))
			Expect(stdoutInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(stdoutInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			// Stderr Log File
			Expect(stderr.Name()).To(Equal(bpmCfg.Stderr()))
			stderrInfo, err := stderr.Stat()
			Expect(err).NotTo(HaveOccurred())
			Expect(stderrInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0600)))
			Expect(stderrInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(stderrInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			// Data Directory
			dataDirInfo, err := os.Stat(bpmCfg.DataDir())
			Expect(err).NotTo(HaveOccurred())
			Expect(dataDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(dataDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(dataDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			// Store Directory
			Expect(bpmCfg.StoreDir()).NotTo(BeADirectory())
			Expect(bpmCfg.StoreDir()).NotTo(BeAnExistingFile())

			// TMP Directory
			tmpDirInfo, err := os.Stat(bpmCfg.TempDir())
			Expect(err).NotTo(HaveOccurred())
			Expect(tmpDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(tmpDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(tmpDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			//Volumes
			for _, vol := range procCfg.Volumes {
				volDirInfo, err := os.Stat(vol)
				Expect(err).NotTo(HaveOccurred())
				Expect(volDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
				Expect(volDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
				Expect(volDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))
			}
		})

		Context("when there is a persistent store", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(systemRoot, "store"), 0700)).To(Succeed())
			})

			It("creates the job prerequisites", func() {
				_, _, err := runcAdapter.CreateJobPrerequisites(bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())

				// Store Directory
				storeDirInfo, err := os.Stat(bpmCfg.StoreDir())
				Expect(err).NotTo(HaveOccurred())
				Expect(storeDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
				Expect(storeDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
				Expect(storeDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))
			})
		})
	})

	Describe("BuildSpec", func() {
		BeforeEach(func() {
			procCfg = &config.ProcessConfig{
				Executable: "/var/vcap/packages/example/bin/example",
				Args: []string{
					"foo",
					"bar",
				},
				Env: []string{
					"RAVE=true",
					"ONE=two",
				},
				Volumes: []string{
					"/path/to/volume/1",
					"/path/to/volume/2",
				},
			}
		})

		It("converts a bpm config into a runc spec", func() {
			spec, err := runcAdapter.BuildSpec(bpmCfg, procCfg, user)
			Expect(err).NotTo(HaveOccurred())

			Expect(spec.Version).To(Equal(specs.Version))

			Expect(spec.Platform).To(Equal(specs.Platform{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
			}))

			expectedProcessArgs := append([]string{procCfg.Executable}, procCfg.Args...)
			Expect(spec.Process).To(Equal(&specs.Process{
				Terminal:        false,
				ConsoleSize:     nil,
				User:            user,
				Args:            expectedProcessArgs,
				Env:             append(procCfg.Env, fmt.Sprintf("TMPDIR=%s", bpmCfg.TempDir())),
				Cwd:             "/",
				Rlimits:         []specs.LinuxRlimit{},
				NoNewPrivileges: true,
			}))

			Expect(spec.Root).To(Equal(specs.Root{
				Path: bpmCfg.RootFSPath(),
			}))

			Expect(spec.Mounts).To(ConsistOf([]specs.Mount{
				{
					Destination: "/proc",
					Type:        "proc",
					Source:      "proc",
					Options:     nil,
				},
				{
					Destination: "/dev",
					Type:        "tmpfs",
					Source:      "tmpfs",
					Options:     []string{"nosuid", "noexec", "mode=755", "size=65536k"},
				},
				{
					Destination: "/dev/pts",
					Type:        "devpts",
					Source:      "devpts",
					Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620", "gid=5"},
				},
				{
					Destination: "/dev/shm",
					Type:        "tmpfs",
					Source:      "shm",
					Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
				},
				{
					Destination: "/dev/mqueue",
					Type:        "mqueue",
					Source:      "mqueue",
					Options:     []string{"nosuid", "noexec", "nodev"},
				},
				{
					Destination: "/sys",
					Type:        "sysfs",
					Source:      "sysfs",
					Options:     []string{"nosuid", "noexec", "nodev", "ro"},
				},
				{
					Destination: "/bin",
					Type:        "bind",
					Source:      "/bin",
					Options:     []string{"nosuid", "nodev", "rbind", "ro"},
				},
				{
					Destination: "/etc",
					Type:        "bind",
					Source:      "/etc",
					Options:     []string{"nosuid", "nodev", "rbind", "ro"},
				},
				{
					Destination: "/usr",
					Type:        "bind",
					Source:      "/usr",
					Options:     []string{"nosuid", "nodev", "rbind", "ro"},
				},
				{
					Destination: "/lib",
					Type:        "bind",
					Source:      "/lib",
					Options:     []string{"nosuid", "nodev", "rbind", "ro"},
				},
				{
					Destination: "/lib64",
					Type:        "bind",
					Source:      "/lib64",
					Options:     []string{"nosuid", "nodev", "rbind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "data", "example"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "example"),
					Options:     []string{"rbind", "rw"},
				},
				{
					Destination: filepath.Join(systemRoot, "data", "packages"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "packages"),
					Options:     []string{"rbind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "jobs", "example"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "jobs", "example"),
					Options:     []string{"rbind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "packages"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "packages"),
					Options:     []string{"rbind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "sys", "log", jobName),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "sys", "log", jobName),
					Options:     []string{"rbind", "rw"},
				},
				{
					Destination: "/path/to/volume/1",
					Type:        "bind",
					Source:      "/path/to/volume/1",
					Options:     []string{"rbind", "rw"},
				},
				{
					Destination: "/path/to/volume/2",
					Type:        "bind",
					Source:      "/path/to/volume/2",
					Options:     []string{"rbind", "rw"},
				},
				{
					Destination: "/tmp",
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "example", "tmp"),
					Options:     []string{"rbind", "rw"},
				},
			}))

			Expect(spec.Linux.RootfsPropagation).To(Equal("private"))
			Expect(spec.Linux.MaskedPaths).To(ConsistOf([]string{
				"/etc/sv",
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
			}))

			Expect(spec.Linux.ReadonlyPaths).To(ConsistOf([]string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			}))

			Expect(spec.Linux.Namespaces).To(ConsistOf(
				specs.LinuxNamespace{Type: "ipc"},
				specs.LinuxNamespace{Type: "mount"},
				specs.LinuxNamespace{Type: "pid"},
				specs.LinuxNamespace{Type: "uts"},
			))
		})

		Context("when there is a persistent store", func() {
			BeforeEach(func() {
				Expect(os.MkdirAll(filepath.Join(systemRoot, "store"), 0700)).To(Succeed())
			})

			It("creates the job prerequisites", func() {
				spec, err := runcAdapter.BuildSpec(bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())

				Expect(spec.Mounts).To(ContainElement(specs.Mount{
					Destination: filepath.Join(systemRoot, "store", "example"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "store", "example"),
					Options:     []string{"rbind", "rw"},
				}))
			})
		})

		Context("Limits", func() {
			BeforeEach(func() {
				procCfg.Limits = &config.Limits{}
			})

			It("sets no limits by default", func() {
				_, err := runcAdapter.BuildSpec(bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("Memory", func() {
				var expectedMemoryLimit string

				BeforeEach(func() {
					expectedMemoryLimit = "100G"
					procCfg.Limits.Memory = &expectedMemoryLimit
				})

				It("sets the memory limit on the container", func() {
					spec, err := runcAdapter.BuildSpec(bpmCfg, procCfg, user)
					Expect(err).NotTo(HaveOccurred())

					expectedMemoryLimitInBytes, err := bytefmt.ToBytes(expectedMemoryLimit)
					Expect(err).NotTo(HaveOccurred())
					Expect(spec.Linux.Resources.Memory).To(Equal(&specs.LinuxMemory{
						Limit: &expectedMemoryLimitInBytes,
						Swap:  &expectedMemoryLimitInBytes,
					}))
				})

				Context("when the memory limit is invalid", func() {
					BeforeEach(func() {
						memoryLimit := "invalid byte value"
						procCfg.Limits.Memory = &memoryLimit
					})

					It("returns an error", func() {
						_, err := runcAdapter.BuildSpec(bpmCfg, procCfg, user)
						Expect(err).To(HaveOccurred())
					})
				})
			})

			Context("OpenFiles", func() {
				var expectedOpenFilesLimit uint64

				BeforeEach(func() {
					expectedOpenFilesLimit = 2444
					procCfg.Limits.OpenFiles = &expectedOpenFilesLimit
				})

				It("sets the rlimit on the process", func() {
					spec, err := runcAdapter.BuildSpec(bpmCfg, procCfg, user)
					Expect(err).NotTo(HaveOccurred())

					Expect(spec.Process.Rlimits).To(ConsistOf([]specs.LinuxRlimit{
						{
							Type: "RLIMIT_NOFILE",
							Hard: uint64(expectedOpenFilesLimit),
							Soft: uint64(expectedOpenFilesLimit),
						},
					}))
				})
			})

			Context("Pids", func() {
				var pidLimit int64

				BeforeEach(func() {
					pidLimit = int64(30)
					procCfg.Limits.Processes = &pidLimit
				})

				It("sets a PidLimit on the container", func() {
					spec, err := runcAdapter.BuildSpec(bpmCfg, procCfg, user)
					Expect(err).NotTo(HaveOccurred())

					Expect(spec.Linux).NotTo(BeNil())
					Expect(spec.Linux.Resources).NotTo(BeNil())
					Expect(spec.Linux.Resources.Pids).NotTo(BeNil())
					Expect(*spec.Linux.Resources.Pids).To(Equal(specs.LinuxPids{
						Limit: pidLimit,
					}))
				})
			})
		})

		Context("when the limits configuration is not provided", func() {
			BeforeEach(func() {
				procCfg.Limits = nil
			})

			It("does not set a memory limit", func() {
				spec, err := runcAdapter.BuildSpec(bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())
				Expect(spec.Linux.Resources).To(BeNil())
			})
		})
	})
})
