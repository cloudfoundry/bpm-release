package runcadapter

import (
	"crucible/config"
	"errors"
	"path/filepath"
	"runtime"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (a *runcAdapter) BuildSpec(jobName string, cfg *config.CrucibleConfig) (specs.Spec, error) {
	user, err := a.userIDFinder.Lookup("vcap")
	if err != nil {
		return specs.Spec{}, err
	}

	if cfg.Process == nil {
		return specs.Spec{}, errors.New("no process defined")
	}

	process := &specs.Process{
		User: user,
		Args: append([]string{cfg.Process.Executable}, cfg.Process.Args...),
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
	}

	mounts := append(defaultMounts(), boshMounts(jobName)...)

	return specs.Spec{
		Version: specs.Version,
		Platform: specs.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		Process: process,
		Root: specs.Root{
			Path: filepath.Join(config.BundlesRoot(), jobName, "rootfs"),
		},
		Hostname: jobName,
		Mounts:   mounts,
		Linux: &specs.Linux{
			RootfsPropagation: "private",
			MaskedPaths: []string{
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
			},
			ReadonlyPaths: []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
			Namespaces: []specs.LinuxNamespace{
				{Type: "uts"},
				{Type: "mount"},
			},
		},
	}, nil
}

func boshMounts(jobName string) []specs.Mount {
	return []specs.Mount{
		{
			Destination: filepath.Join(config.BoshRoot(), "jobs", jobName),
			Type:        "bind",
			Source:      filepath.Join(config.BoshRoot(), "jobs", jobName),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: filepath.Join(config.BoshRoot(), "data", "packages"),
			Type:        "bind",
			Source:      filepath.Join(config.BoshRoot(), "data", "packages"),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: filepath.Join(config.BoshRoot(), "packages"),
			Type:        "bind",
			Source:      filepath.Join(config.BoshRoot(), "packages"),
			Options:     []string{"rbind", "ro"},
		},
	}
}

func defaultMounts() []specs.Mount {
	return []specs.Mount{
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
	}
}
