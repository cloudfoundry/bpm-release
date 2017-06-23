package runcadapter

import (
	"crucible/config"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"code.cloudfoundry.org/bytefmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const ROOT_UID = 0

//go:generate counterfeiter . RuncAdapter

type RuncAdapter interface {
	CreateJobPrerequisites(systemRoot, jobName string, cfg *config.CrucibleConfig, user specs.User) (string, *os.File, *os.File, error)
	BuildSpec(systemRoot, jobName string, cfg *config.CrucibleConfig, user specs.User) (specs.Spec, error)
}

type runcAdapter struct{}

func NewRuncAdapter() RuncAdapter {
	return &runcAdapter{}
}

func (a *runcAdapter) CreateJobPrerequisites(
	systemRoot string,
	jobName string,
	cfg *config.CrucibleConfig,
	user specs.User,
) (string, *os.File, *os.File, error) {
	cruciblePidDir := filepath.Join(systemRoot, "sys", "run", "crucible", jobName)
	jobLogDir := filepath.Join(systemRoot, "sys", "log", jobName)
	stdoutFileLocation := filepath.Join(jobLogDir, fmt.Sprintf("%s.out.log", cfg.Name))
	stderrFileLocation := filepath.Join(jobLogDir, fmt.Sprintf("%s.err.log", cfg.Name))
	dataDir := filepath.Join(systemRoot, "data", jobName, cfg.Name)

	err := os.MkdirAll(cruciblePidDir, 0700)
	if err != nil {
		return "", nil, nil, err
	}

	err = os.MkdirAll(jobLogDir, 0750)
	if err != nil {
		return "", nil, nil, err
	}
	err = os.Chown(jobLogDir, ROOT_UID, int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	stdout, err := createFileFor(stdoutFileLocation, int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	stderr, err := createFileFor(stderrFileLocation, int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	err = os.MkdirAll(dataDir, 0700)
	if err != nil {
		return "", nil, nil, err
	}
	err = os.Chown(dataDir, int(user.UID), int(user.GID))
	if err != nil {
		return "", nil, nil, err
	}

	return cruciblePidDir, stdout, stderr, nil
}

func createFileFor(path string, uid, gid int) (*os.File, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0700)
	if err != nil {
		return nil, err
	}

	err = os.Chown(path, uid, gid)
	if err != nil {
		return nil, err
	}

	return f, nil
}

func (a *runcAdapter) BuildSpec(
	systemRoot string,
	jobName string,
	cfg *config.CrucibleConfig,
	user specs.User,
) (specs.Spec, error) {
	process := &specs.Process{
		User: user,
		Args: append([]string{cfg.Executable}, cfg.Args...),
		Env:  cfg.Env,
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

	mounts := defaultMounts()
	mounts = append(mounts, boshMounts(systemRoot, jobName, cfg.Name)...)
	mounts = append(mounts, systemIdentityMounts()...)

	var resources *specs.LinuxResources
	if cfg.Limits != nil {
		memLimit, err := bytefmt.ToBytes(cfg.Limits.Memory)
		if err != nil {
			return specs.Spec{}, err
		}

		falsePtr := false
		resources = &specs.LinuxResources{
			DisableOOMKiller: &falsePtr,
			Memory: &specs.LinuxMemory{
				Limit: &memLimit,
			},
		}
	}

	return specs.Spec{
		Version: specs.Version,
		Platform: specs.Platform{
			OS:   runtime.GOOS,
			Arch: runtime.GOARCH,
		},
		Process: process,
		Root: specs.Root{
			Path: filepath.Join(config.BundlesRoot(), jobName, cfg.Name, "rootfs"),
		},
		Hostname: jobName,
		Mounts:   mounts,
		Linux: &specs.Linux{
			MaskedPaths: []string{
				"/proc/kcore",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
			},
			Namespaces: []specs.LinuxNamespace{
				{Type: "uts"},
				{Type: "mount"},
				{Type: "pid"},
			},
			ReadonlyPaths: []string{
				"/proc/asound",
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
			Resources:         resources,
			RootfsPropagation: "private",
		},
	}, nil
}

func boshMounts(systemRoot, jobName, procName string) []specs.Mount {
	return []specs.Mount{
		{
			Destination: filepath.Join(systemRoot, "data", jobName, procName),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "data", jobName, procName),
			Options:     []string{"rbind", "rw"},
		},
		{
			Destination: filepath.Join(systemRoot, "data", "packages"),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "data", "packages"),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: filepath.Join(systemRoot, "jobs", jobName),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "jobs", jobName),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: filepath.Join(systemRoot, "packages"),
			Type:        "bind",
			Source:      filepath.Join(systemRoot, "packages"),
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
	}
}

func systemIdentityMounts() []specs.Mount {
	return []specs.Mount{
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
