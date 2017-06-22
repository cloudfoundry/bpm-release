package runcadapter_test

import (
	"crucible/config"
	"crucible/runcadapter"
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
		adapter runcadapter.RuncAdapter

		jobName,
		systemRoot string
		jobConfig *config.CrucibleConfig
		user      specs.User
	)

	BeforeEach(func() {
		adapter = runcadapter.NewRuncAdapter()

		jobName = "example"
		jobConfig = &config.CrucibleConfig{
			Name:       "server",
			Executable: "executable",
		}
		user = specs.User{UID: 200, GID: 300, Username: "vcap"}

		var err error
		systemRoot, err = ioutil.TempDir("", "runc-adapter-system-files")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(systemRoot)).To(Succeed())
	})

	Describe("CreateJobPrerequisites", func() {
		It("creates the job prerequisites", func() {
			pidDir, stdout, stderr, err := adapter.CreateJobPrerequisites(systemRoot, jobName, jobConfig, user)
			Expect(err).NotTo(HaveOccurred())

			logDir := filepath.Join(systemRoot, "sys", "log", jobName)
			expectedStdoutFileName := fmt.Sprintf("%s.out.log", jobConfig.Name)
			expectedStderrFileName := fmt.Sprintf("%s.err.log", jobConfig.Name)

			// PID Directory
			Expect(pidDir).To(Equal(filepath.Join(systemRoot, "sys", "run", "crucible", jobName)))

			// Log Directory
			logDirInfo, err := os.Stat(logDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(logDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0750)))
			Expect(logDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(0)))
			Expect(logDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			// Stdout Log File
			Expect(stdout.Name()).To(Equal(filepath.Join(logDir, expectedStdoutFileName)))
			stdoutInfo, err := stdout.Stat()
			Expect(err).NotTo(HaveOccurred())
			Expect(stdoutInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(stdoutInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(stdoutInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			// Stderr Log File
			Expect(stderr.Name()).To(Equal(filepath.Join(logDir, expectedStderrFileName)))
			stderrInfo, err := stderr.Stat()
			Expect(err).NotTo(HaveOccurred())
			Expect(stderrInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(stderrInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(stderrInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))

			// Data Directory
			dataDir := filepath.Join(systemRoot, "data", jobName, jobConfig.Name)
			dataDirInfo, err := os.Stat(dataDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(dataDirInfo.Mode() & os.ModePerm).To(Equal(os.FileMode(0700)))
			Expect(dataDirInfo.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(200)))
			Expect(dataDirInfo.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(300)))
		})
	})

	Describe("BuildSpec", func() {
		var cfg *config.CrucibleConfig

		BeforeEach(func() {
			cfg = &config.CrucibleConfig{
				Name:       "server",
				Executable: "/var/vcap/packages/example/bin/example",
				Args: []string{
					"foo",
					"bar",
				},
				Env: []string{
					"RAVE=true",
					"ONE=two",
				},
			}
		})

		It("convert a crucible config into a runc spec", func() {
			spec := adapter.BuildSpec(systemRoot, jobName, cfg, user)

			Expect(spec.Version).To(Equal(specs.Version))

			Expect(spec.Platform).To(Equal(specs.Platform{
				OS:   runtime.GOOS,
				Arch: runtime.GOARCH,
			}))

			expectedProcessArgs := append([]string{cfg.Executable}, cfg.Args...)
			Expect(spec.Process).To(Equal(&specs.Process{
				Terminal:    false,
				ConsoleSize: nil,
				User:        user,
				Args:        expectedProcessArgs,
				Env:         cfg.Env,
				Cwd:         "/",
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
				Path: "/var/vcap/data/crucible/bundles/example/server/rootfs",
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
					Destination: filepath.Join(systemRoot, "data", "example", "server"),
					Type:        "bind",
					Source:      filepath.Join(systemRoot, "data", "example", "server"),
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

			Expect(spec.Linux.Namespaces).To(ConsistOf(
				specs.LinuxNamespace{Type: "uts"},
				specs.LinuxNamespace{Type: "mount"},
				specs.LinuxNamespace{Type: "pid"},
			))
		})
	})
})
