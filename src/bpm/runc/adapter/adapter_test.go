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

package adapter_test

import (
	"bpm/config"
	"bpm/runc/adapter"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"code.cloudfoundry.org/bytefmt"
	"code.cloudfoundry.org/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/opencontainers/runtime-spec/specs-go"
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
		logger  *lagertest.TestLogger
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("adapter")
		runcAdapter = adapter.NewRuncAdapter()

		jobName = "example"
		procName = "server"
		user = specs.User{UID: 200, GID: 300, Username: "vcap"}

		var err error
		systemRoot, err = ioutil.TempDir("", "runc-adapter-system-files")
		Expect(err).NotTo(HaveOccurred())

		bpmCfg = config.NewBPMConfig(systemRoot, jobName, procName)
		procCfg = &config.ProcessConfig{
			AdditionalVolumes: []config.Volume{
				{Path: filepath.Join(systemRoot, "some", "directory")},
				{Path: filepath.Join(systemRoot, "another", "location")},
			},
		}

		Expect(os.MkdirAll(filepath.Join(systemRoot, "store"), 0700)).To(Succeed())
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

			// Data Directory should not be writable
			dataDirInfo, err := os.Stat(bpmCfg.DataDir())
			Expect(err).NotTo(HaveOccurred())
			Expect(dataDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(dataDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
			Expect(dataDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(0)))

			// Store Directory
			Expect(bpmCfg.StoreDir()).NotTo(BeADirectory())
			Expect(bpmCfg.StoreDir()).NotTo(BeAnExistingFile())

			// TMP Directory
			tmpDirInfo, err := os.Stat(bpmCfg.TempDir())
			Expect(err).NotTo(HaveOccurred())
			Expect(tmpDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(tmpDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(tmpDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			//AdditionalVolumes
			for _, vol := range procCfg.AdditionalVolumes {
				volDirInfo, err := os.Stat(vol.Path)
				Expect(err).NotTo(HaveOccurred())
				Expect(volDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
				Expect(volDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
				Expect(volDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))
			}
		})

		Context("when the user requests a persistent disk", func() {
			BeforeEach(func() {
				procCfg.PersistentDisk = true
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

			Context("and the persistent disk directory does not exist", func() {
				BeforeEach(func() {
					Expect(os.RemoveAll(filepath.Join(systemRoot, "store"))).To(Succeed())
				})

				It("creates the job prerequisites", func() {
					_, _, err := runcAdapter.CreateJobPrerequisites(bpmCfg, procCfg, user)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when the user requests an ephemeral disk", func() {
			BeforeEach(func() {
				procCfg.EphemeralDisk = true
			})

			It("creates the data directory with the correct permissions", func() {
				_, _, err := runcAdapter.CreateJobPrerequisites(bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())

				// Data Directory
				dataDirInfo, err := os.Stat(bpmCfg.DataDir())
				Expect(err).NotTo(HaveOccurred())
				Expect(dataDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
				Expect(dataDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
				Expect(dataDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))
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
				Env: map[string]string{
					"RAVE": "true",
					"ONE":  "two",
				},
				AdditionalVolumes: []config.Volume{
					{Path: "/path/to/volume/1", Writable: true},
					{Path: "/path/to/volume/jna-tmp", Writable: true, AllowExecutions: true},
					// Testing duplicate volumes
					{Path: "/path/to/volume/2"},
					{Path: "/path/to/volume/2"},
					// Testing duplicate store mount and ignore writable flag
					{Path: bpmCfg.DataDir()},
					// Testing duplicate data mount
					{Path: bpmCfg.StoreDir(), Writable: true},
				},
				Capabilities: []string{"TAIN", "SAICIN"},
			}
		})

		convertEnv := func(env map[string]string) []string {
			var environ []string

			for k, v := range env {
				environ = append(environ, fmt.Sprintf("%s=%s", k, v))
			}

			return environ
		}

		It("converts a bpm config into a runc spec", func() {
			spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
			Expect(err).NotTo(HaveOccurred())

			Expect(spec.Version).To(Equal(specs.Version))

			expectedProcessArgs := append([]string{procCfg.Executable}, procCfg.Args...)
			expectedEnv := convertEnv(procCfg.Env)
			expectedEnv = append(expectedEnv, fmt.Sprintf("TMPDIR=%s", bpmCfg.TempDir()))
			expectedEnv = append(expectedEnv, fmt.Sprintf("LANG=%s", adapter.DefaultLang))
			expectedEnv = append(expectedEnv, fmt.Sprintf("PATH=%s", adapter.DefaultPath(bpmCfg)))
			expectedEnv = append(expectedEnv, fmt.Sprintf("HOME=%s", bpmCfg.DataDir()))

			Expect(spec.Process.Terminal).To(Equal(false))
			Expect(spec.Process.ConsoleSize).To(BeNil())
			Expect(spec.Process.User).To(Equal(user))
			Expect(spec.Process.Args).To(Equal(expectedProcessArgs))
			Expect(spec.Process.Env).To(ConsistOf(expectedEnv))
			Expect(spec.Process.Cwd).To(Equal(bpmCfg.JobDir()))
			Expect(spec.Process.Rlimits).To(Equal([]specs.POSIXRlimit{}))
			Expect(spec.Process.NoNewPrivileges).To(Equal(true))
			Expect(spec.Process.Capabilities).To(Equal(&specs.LinuxCapabilities{
				Bounding:    []string{"CAP_TAIN", "CAP_SAICIN"},
				Effective:   []string{},
				Inheritable: []string{"CAP_TAIN", "CAP_SAICIN"},
				Permitted:   []string{"CAP_TAIN", "CAP_SAICIN"},
				Ambient:     []string{"CAP_TAIN", "CAP_SAICIN"},
			}))

			Expect(spec.Root).To(Equal(&specs.Root{
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
					Options:     []string{"nosuid", "nodev", "bind", "ro"},
				},
				{
					Destination: "/etc",
					Type:        "bind",
					Source:      "/etc",
					Options:     []string{"nosuid", "nodev", "bind", "ro"},
				},
				{
					Destination: "/usr",
					Type:        "bind",
					Source:      "/usr",
					Options:     []string{"nosuid", "nodev", "bind", "ro"},
				},
				{
					Destination: "/lib",
					Type:        "bind",
					Source:      "/lib",
					Options:     []string{"nosuid", "nodev", "bind", "ro"},
				},
				{
					Destination: "/lib64",
					Type:        "bind",
					Source:      "/lib64",
					Options:     []string{"nosuid", "nodev", "bind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "data", "packages"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "packages"),
					Options:     []string{"nodev", "nosuid", "bind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "jobs", "example"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "jobs", "example"),
					Options:     []string{"nodev", "nosuid", "bind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "packages"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "packages"),
					Options:     []string{"nodev", "nosuid", "bind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "sys", "log", jobName),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "sys", "log", jobName),
					Options:     []string{"nodev", "nosuid", "noexec", "rbind", "rw"},
				},
				{
					Destination: "/path/to/volume/1",
					Type:        "bind",
					Source:      "/path/to/volume/1",
					Options:     []string{"nodev", "nosuid", "noexec", "rbind", "rw"},
				},
				{
					Destination: "/path/to/volume/jna-tmp",
					Type:        "bind",
					Source:      "/path/to/volume/jna-tmp",
					Options:     []string{"nodev", "nosuid", "exec", "rbind", "rw"},
				},
				{
					Destination: "/path/to/volume/2",
					Type:        "bind",
					Source:      "/path/to/volume/2",
					Options:     []string{"nodev", "nosuid", "noexec", "rbind", "ro"},
				},
				{
					Destination: filepath.Join(systemRoot, "data", "example", "tmp"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "example", "tmp"),
					Options:     []string{"nodev", "nosuid", "noexec", "rbind", "rw"},
				},
				{
					Destination: "/var/tmp",
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "example", "tmp"),
					Options:     []string{"nodev", "nosuid", "noexec", "rbind", "rw"},
				},
				{
					Destination: "/tmp",
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "example", "tmp"),
					Options:     []string{"nodev", "nosuid", "noexec", "rbind", "rw"},
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

			// This must be part of the existing It block to preven test pollution
			By("the presence of /run/resolvconf on the host")

			Expect(os.MkdirAll(adapter.ResolvConfDir, 0700)).To(Succeed())

			specWithResolvConf, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
			Expect(err).NotTo(HaveOccurred())

			Expect(specWithResolvConf.Mounts).To(ContainElement(specs.Mount{
				Destination: adapter.ResolvConfDir,
				Type:        "bind",
				Source:      adapter.ResolvConfDir,
				Options:     []string{"nodev", "nosuid", "noexec", "bind", "ro"},
			}))

			Expect(os.RemoveAll(adapter.ResolvConfDir)).To(Succeed())
		})

		Context("when a user provides TMPDIR, LANG and PATH, and HOME environment variables", func() {
			BeforeEach(func() {
				procCfg.Env["TMPDIR"] = "/I/AM/A/TMPDIR"
				procCfg.Env["LANG"] = "esperanto"
				procCfg.Env["PATH"] = "some-path"
				procCfg.Env["HOME"] = "some-home"
			})

			It("uses the user-provided values", func() {
				spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())
				Expect(spec.Process.Env).NotTo(ContainElement(fmt.Sprintf("TMPDIR=%s", bpmCfg.TempDir())))
				Expect(spec.Process.Env).NotTo(ContainElement(fmt.Sprintf("LANG=%s", adapter.DefaultLang)))
				Expect(spec.Process.Env).NotTo(ContainElement(fmt.Sprintf("PATH=%s", adapter.DefaultPath(bpmCfg))))
				Expect(spec.Process.Env).NotTo(ContainElement(fmt.Sprintf("HOME=%s", bpmCfg.DataDir())))
				Expect(spec.Process.Env).To(ContainElement("TMPDIR=/I/AM/A/TMPDIR"))
				Expect(spec.Process.Env).To(ContainElement("LANG=esperanto"))
				Expect(spec.Process.Env).To(ContainElement("PATH=some-path"))
				Expect(spec.Process.Env).To(ContainElement("HOME=some-home"))
			})
		})

		Context("when a workdir is provided", func() {
			BeforeEach(func() {
				procCfg.WorkDir = "/I/AM/A/WORKDIR"
			})

			It("sets the current working directory of the process", func() {
				spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())
				Expect(spec.Process.Cwd).To(Equal("/I/AM/A/WORKDIR"))
			})
		})

		Context("when the user requests a persistent disk", func() {
			BeforeEach(func() {
				procCfg.PersistentDisk = true
			})

			It("bind mounts the store directory into the container", func() {
				spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())

				Expect(spec.Mounts).To(ContainElement(specs.Mount{
					Destination: filepath.Join(systemRoot, "store", "example"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "store", "example"),
					Options:     []string{"nodev", "nosuid", "noexec", "rbind", "rw"},
				}))
			})
		})

		Context("when the user requests an ephemeral disk", func() {
			BeforeEach(func() {
				procCfg.EphemeralDisk = true
			})

			It("bind mounts the data directory into the container", func() {
				spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())

				Expect(spec.Mounts).To(ContainElement(specs.Mount{
					Destination: filepath.Join(systemRoot, "data", "example"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "example"),
					Options:     []string{"nodev", "nosuid", "noexec", "rbind", "rw"},
				}))
			})
		})

		Context("Limits", func() {
			BeforeEach(func() {
				procCfg.Limits = &config.Limits{}
			})

			It("sets no limits by default", func() {
				_, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("Memory", func() {
				var expectedMemoryLimit string

				BeforeEach(func() {
					expectedMemoryLimit = "100G"
					procCfg.Limits.Memory = &expectedMemoryLimit
				})

				It("sets the memory limit on the container", func() {
					spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
					Expect(err).NotTo(HaveOccurred())

					expectedMemoryLimitInBytes, err := bytefmt.ToBytes(expectedMemoryLimit)
					Expect(err).NotTo(HaveOccurred())
					signedExpectedMemoryLimitInBytes := int64(expectedMemoryLimitInBytes)
					Expect(spec.Linux.Resources.Memory).To(Equal(&specs.LinuxMemory{
						Limit: &signedExpectedMemoryLimitInBytes,
						Swap:  &signedExpectedMemoryLimitInBytes,
					}))
				})

				Context("when the memory limit is invalid", func() {
					BeforeEach(func() {
						memoryLimit := "invalid byte value"
						procCfg.Limits.Memory = &memoryLimit
					})

					It("returns an error", func() {
						_, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
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
					spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
					Expect(err).NotTo(HaveOccurred())

					Expect(spec.Process.Rlimits).To(ConsistOf([]specs.POSIXRlimit{
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
					spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
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
				spec, err := runcAdapter.BuildSpec(logger, bpmCfg, procCfg, user)
				Expect(err).NotTo(HaveOccurred())
				Expect(spec.Linux.Resources).To(BeNil())
			})
		})
	})
})
