package runcadapter

import (
	"crucible/config"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const ROOT_UID = 0

//go:generate counterfeiter . RuncAdapter

type RuncAdapter interface {
	CreateJobPrerequisites(systemRoot, jobName, procName string) (string, *os.File, *os.File, error)
	BuildSpec(jobName string, jobConfig *config.CrucibleConfig) (specs.Spec, error)
	CreateBundle(bundlesRoot, jobName, procName string, jobSpec specs.Spec) (string, error)
	RunContainer(pidFilePath, bundlePath, containerID string, stdout, stderr io.Writer) error
	ContainerState(containerID string) (specs.State, error)
	StopContainer(containerID string) error
	DeleteContainer(containerID string) error
	DestroyBundle(bundlesRoot, jobName, procName string) error
}

type runcAdapter struct {
	runcPath     string
	userIDFinder UserIDFinder
}

func NewRuncAdapter(runcPath string, userIDFinder UserIDFinder) RuncAdapter {
	return &runcAdapter{
		runcPath:     runcPath,
		userIDFinder: userIDFinder,
	}
}

func (a *runcAdapter) CreateJobPrerequisites(systemRoot, jobName, procName string) (string, *os.File, *os.File, error) {
	cruciblePidDir := filepath.Join(systemRoot, "sys", "run", "crucible", jobName)
	err := os.MkdirAll(cruciblePidDir, 0700)
	if err != nil {
		return "", nil, nil, err
	}

	user, err := a.userIDFinder.Lookup("vcap")
	if err != nil {
		return "", nil, nil, err
	}

	jobLogDir := filepath.Join(systemRoot, "sys", "log", jobName)
	stdoutFileLocation := filepath.Join(jobLogDir, fmt.Sprintf("%s.out.log", procName))
	stderrFileLocation := filepath.Join(jobLogDir, fmt.Sprintf("%s.err.log", procName))

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

	dataDir := filepath.Join(systemRoot, "data", jobName, procName)
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

func (a *runcAdapter) BuildSpec(jobName string, cfg *config.CrucibleConfig) (specs.Spec, error) {
	user, err := a.userIDFinder.Lookup("vcap")
	if err != nil {
		return specs.Spec{}, err
	}

	if cfg.Name == "" || cfg.Executable == "" {
		return specs.Spec{}, errors.New("no process defined")
	}

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
	mounts = append(mounts, boshMounts(jobName, cfg.Name)...)
	mounts = append(mounts, systemIdentityMounts()...)

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
				{Type: "pid"},
			},
		},
	}, nil
}

func (a *runcAdapter) CreateBundle(bundlesRoot, jobName, procName string, jobSpec specs.Spec) (string, error) {
	bundlePath := filepath.Join(bundlesRoot, jobName, procName)
	err := os.MkdirAll(bundlePath, 0700)
	if err != nil {
		return "", err
	}
	rootfsPath := filepath.Join(bundlePath, "rootfs")
	err = os.MkdirAll(rootfsPath, 0700)
	if err != nil {
		return "", err
	}

	user, err := a.userIDFinder.Lookup("vcap") // hardcoded for now
	if err != nil {
		return "", err
	}

	err = os.Chown(rootfsPath, int(user.UID), int(user.GID))
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(filepath.Join(bundlePath, "config.json"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		// This is super hard to test as we are root.
		return "", err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	err = enc.Encode(&jobSpec)
	if err != nil {
		// Hard to test - spec was defined by golang so this should not be invalid json
		return "", err
	}

	return bundlePath, nil
}

func (a *runcAdapter) RunContainer(pidFilePath, bundlePath, containerID string, stdout, stderr io.Writer) error {
	runcCmd := exec.Command(
		a.runcPath,
		"run",
		"--bundle", bundlePath,
		"--pid-file", pidFilePath,
		"--detach",
		containerID,
	)

	runcCmd.Stdout = stdout
	runcCmd.Stderr = stderr

	return runcCmd.Run()
}

func (a *runcAdapter) ContainerState(containerID string) (specs.State, error) {
	runcCmd := exec.Command(
		a.runcPath,
		"state",
		containerID,
	)

	var state specs.State
	data, err := runcCmd.CombinedOutput()
	if err != nil {
		return specs.State{}, err
	}

	err = json.Unmarshal(data, &state)
	if err != nil {
		return specs.State{}, err
	}

	return state, nil
}

func (a *runcAdapter) StopContainer(containerID string) error {
	runcCmd := exec.Command(
		a.runcPath,
		"kill",
		containerID,
	)

	return runcCmd.Run()
}

func (a *runcAdapter) DeleteContainer(containerID string) error {
	runcCmd := exec.Command(
		a.runcPath,
		"delete",
		"-f",
		containerID,
	)

	return runcCmd.Run()
}

func (a *runcAdapter) DestroyBundle(bundlesRoot, jobName, procName string) error {
	return os.RemoveAll(filepath.Join(bundlesRoot, jobName, procName))
}

func boshMounts(jobName, procName string) []specs.Mount {
	return []specs.Mount{
		{
			Destination: filepath.Join(config.BoshRoot(), "data", jobName, procName),
			Type:        "bind",
			Source:      filepath.Join(config.BoshRoot(), "data", jobName, procName),
			Options:     []string{"rbind", "rw"},
		},
		{
			Destination: filepath.Join(config.BoshRoot(), "data", "packages"),
			Type:        "bind",
			Source:      filepath.Join(config.BoshRoot(), "data", "packages"),
			Options:     []string{"rbind", "ro"},
		},
		{
			Destination: filepath.Join(config.BoshRoot(), "jobs", jobName),
			Type:        "bind",
			Source:      filepath.Join(config.BoshRoot(), "jobs", jobName),
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
