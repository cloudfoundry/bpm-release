package specbuilder

import (
	"crucible/config"
	"fmt"
	"runtime"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

//go:generate counterfeiter . UserIDFinder
type UserIDFinder interface {
	Lookup(username string) (specs.User, error)
}

func Build(jobName string, cfg *config.CrucibleConfig, idFinder UserIDFinder) specs.Spec {
	user, _ := idFinder.Lookup("vcap")

	process := &specs.Process{
		User: user,
		Args: cfg.Process.Args,
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
			Path: fmt.Sprintf("/var/vcap/data/crucible/%s/rootfs", jobName),
		},
		Hostname: jobName,
		Mounts:   mounts,
	}
}

func boshMounts(jobName string) []specs.Mount {
	return []specs.Mount{
		{
			Destination: fmt.Sprintf("/var/vcap/jobs/%s", jobName),
			Type:        "bind",
			Source:      fmt.Sprintf("/var/vcap/jobs/%s", jobName),
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
