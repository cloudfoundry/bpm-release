package runcadapter_test

import (
	"crucible/config"
	"crucible/runcadapter"
	"crucible/runcadapter/runcadapterfakes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("RuncAdapter", func() {
	var (
		adapter          runcadapter.RuncAdapter
		jobName          string
		fakeUserIDFinder *runcadapterfakes.FakeUserIDFinder
		jobSpec          specs.Spec
	)

	BeforeEach(func() {
		fakeUserIDFinder = &runcadapterfakes.FakeUserIDFinder{}
		fakeUserIDFinder.LookupReturns(specs.User{UID: 200, GID: 300, Username: "vcap"}, nil)

		adapter = runcadapter.NewRuncAdapter("/var/vcap/packages/runc/bin/runc", fakeUserIDFinder)
		jobName = "example"
	})

	Describe("BuildSpec", func() {
		var cfg *config.CrucibleConfig

		BeforeEach(func() {
			cfg = &config.CrucibleConfig{
				Process: &config.Process{
					Executable: "/var/vcap/packages/example/bin/example",
					Args: []string{
						"foo",
						"bar",
					},
					Env: []string{
						"RAVE=true",
						"ONE=two",
					},
				},
			}
		})

		It("convert a crucible config into a runc spec", func() {
			spec, err := adapter.BuildSpec(jobName, cfg)
			Expect(err).NotTo(HaveOccurred())

			Expect(spec.Version).To(Equal(specs.Version))

			Expect(spec.Platform).To(Equal(specs.Platform{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
			}))

			expectedProcessArgs := append([]string{cfg.Process.Executable}, cfg.Process.Args...)
			Expect(fakeUserIDFinder.LookupCallCount()).To(Equal(1))
			Expect(fakeUserIDFinder.LookupArgsForCall(0)).To(Equal("vcap"))
			Expect(spec.Process).To(Equal(&specs.Process{
				Terminal:    false,
				ConsoleSize: nil,
				User: specs.User{
					UID:      200,
					GID:      300,
					Username: "vcap",
				},
				Args: expectedProcessArgs,
				Env:  cfg.Process.Env,
				Cwd:  "/",
				Rlimits: []specs.LinuxRlimit{
					{
						Type: "RLIMIT_NOFILE",
						Hard: uint64(1024),
						Soft: uint64(1024),
					},
				},
				NoNewPrivileges: true,
			}))

			Expect(spec.Root).To(Equal(specs.Root{
				Path: "/var/vcap/data/crucible/bundles/example/rootfs",
			}))

			Expect(spec.Hostname).To(Equal("example"))

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
					Destination: "/sys/fs/cgroup",
					Type:        "cgroup",
					Source:      "cgroup",
					Options:     []string{"nosuid", "noexec", "nodev", "relatime", "ro"},
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
					Destination: "/var/vcap/jobs/example",
					Type:        "bind",
					Source:      "/var/vcap/jobs/example",
					Options:     []string{"rbind", "ro"},
				},
				{
					Destination: "/var/vcap/data/packages",
					Type:        "bind",
					Source:      "/var/vcap/data/packages",
					Options:     []string{"rbind", "ro"},
				},
				{
					Destination: "/var/vcap/packages",
					Type:        "bind",
					Source:      "/var/vcap/packages",
					Options:     []string{"rbind", "ro"},
				},
			}))

			Expect(spec.Linux.RootfsPropagation).To(Equal("private"))
			Expect(spec.Linux.MaskedPaths).To(ConsistOf([]string{
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

			Expect(spec.Linux.Namespaces).To(ConsistOf([]specs.LinuxNamespace{{Type: "uts"}, {Type: "mount"}}))
		})

		Context("when the user id lookup fails", func() {
			BeforeEach(func() {
				fakeUserIDFinder.LookupReturns(specs.User{}, errors.New("this user does not exist"))
			})

			It("returns an error", func() {
				_, err := adapter.BuildSpec(jobName, cfg)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when there is no process specified", func() {
			BeforeEach(func() {
				cfg = &config.CrucibleConfig{}
			})

			It("returns an error", func() {
				_, err := adapter.BuildSpec(jobName, cfg)
				Expect(err).To(MatchError("no process defined"))
			})
		})
	})

	Context("CreateBundle", func() {
		var bundlesRoot string

		BeforeEach(func() {
			jobConfig := &config.CrucibleConfig{
				Process: &config.Process{
					Executable: "/bin/sleep",
					Args:       []string{"100"},
					Env:        []string{"FOO=BAR"},
				},
			}

			var err error
			jobSpec, err = adapter.BuildSpec(jobName, jobConfig)
			Expect(err).ToNot(HaveOccurred())

			bundlesRoot, err = ioutil.TempDir("", "bundle-builder")
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(bundlesRoot)).To(Succeed())
		})

		It("makes the bundle directory", func() {
			bundlePath, err := adapter.CreateBundle(bundlesRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			f, err := os.Stat(bundlePath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
		})

		It("makes an empty rootfs directory", func() {
			bundlePath, err := adapter.CreateBundle(bundlesRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			bundlefs := filepath.Join(bundlePath, "rootfs")
			f, err := os.Stat(bundlefs)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.IsDir()).To(BeTrue())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(f.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(f.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			infos, err := ioutil.ReadDir(bundlefs)
			Expect(err).ToNot(HaveOccurred())
			Expect(infos).To(HaveLen(0))
		})

		It("writes a config.json in the root bundle directory", func() {
			bundlePath, err := adapter.CreateBundle(bundlesRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())

			configPath := filepath.Join(bundlePath, "config.json")
			f, err := os.Stat(configPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(f.Mode() & os.ModePerm).To(Equal(os.FileMode(0600)))

			expectedConfigData, err := json.MarshalIndent(&jobSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			configData, err := ioutil.ReadFile(configPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(configData).To(MatchJSON(expectedConfigData))
		})

		Context("when creating the bundle directory fails", func() {
			BeforeEach(func() {
				_, err := os.Create(filepath.Join(bundlesRoot, jobName))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				_, err := adapter.CreateBundle(bundlesRoot, jobName, jobSpec)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when creating the rootfs directory fails", func() {
			BeforeEach(func() {
				bundlePath := filepath.Join(bundlesRoot, jobName)
				err := os.MkdirAll(bundlePath, 0700)
				Expect(err).ToNot(HaveOccurred())

				_, err = os.Create(filepath.Join(bundlePath, "rootfs"))
				Expect(err).NotTo(HaveOccurred())
			})

			It("returns the error", func() {
				_, err := adapter.CreateBundle(bundlesRoot, jobName, jobSpec)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("CreateSystemFiles", func() {
		var systemRoot string

		BeforeEach(func() {
			var err error
			systemRoot, err = ioutil.TempDir("", "runc-adapter-system-files")
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(systemRoot)).To(Succeed())
		})

		It("creates the system files", func() {
			pidDir, stdout, stderr, err := adapter.CreateSystemFiles(systemRoot, jobName)
			Expect(err).NotTo(HaveOccurred())

			Expect(fakeUserIDFinder.LookupCallCount()).To(Equal(1))
			Expect(fakeUserIDFinder.LookupArgsForCall(0)).To(Equal("vcap"))

			logDir := filepath.Join(systemRoot, "sys", "log", jobName)
			stdoutFileName := fmt.Sprintf("%s.out.log", jobName)
			stderrFileName := fmt.Sprintf("%s.err.log", jobName)

			Expect(pidDir).To(Equal(filepath.Join(systemRoot, "sys", "run", "crucible")))

			logDirInfo, err := os.Stat(logDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(logDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0750)))
			Expect(logDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
			Expect(logDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			Expect(stdout.Name()).To(Equal(filepath.Join(logDir, stdoutFileName)))
			stdoutInfo, err := stdout.Stat()
			Expect(err).NotTo(HaveOccurred())
			Expect(stdoutInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(stdoutInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(stdoutInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			Expect(stderr.Name()).To(Equal(filepath.Join(logDir, stderrFileName)))
			stderrInfo, err := stderr.Stat()
			Expect(err).NotTo(HaveOccurred())
			Expect(stderrInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(stderrInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(stderrInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))
		})

		Context("when looking up the vcap user fails", func() {
			BeforeEach(func() {
				fakeUserIDFinder.LookupReturns(specs.User{}, errors.New("Boom!"))
			})

			It("returns an error", func() {
				_, _, _, err := adapter.CreateSystemFiles(systemRoot, jobName)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Context("DestroyBundle", func() {
		var bundlesRoot, bundlePath string

		BeforeEach(func() {
			var err error
			bundlesRoot, err = ioutil.TempDir("", "bundle-builder")
			Expect(err).ToNot(HaveOccurred())

			jobSpec := specs.Spec{
				Version: "test-version",
			}

			bundlePath, err = adapter.CreateBundle(bundlesRoot, jobName, jobSpec)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			Expect(os.RemoveAll(bundlesRoot)).To(Succeed())
		})

		It("deletes the bundle", func() {
			err := adapter.DestroyBundle(bundlesRoot, jobName)
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(bundlePath)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})
