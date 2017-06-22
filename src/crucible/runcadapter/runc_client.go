package runcadapter

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

//go:generate counterfeiter . RuncClient

type RuncClient interface {
	CreateBundle(bundlePath string, jobSpec specs.Spec, user specs.User) error
	RunContainer(pidFilePath, bundlePath, containerID string, stdout, stderr io.Writer) error
	ContainerState(containerID string) (specs.State, error)
	StopContainer(containerID string) error
	DeleteContainer(containerID string) error
	DestroyBundle(bundlePath string) error
}

type runcClient struct {
	runcPath string
}

func NewRuncClient(runcPath string) RuncClient {
	return &runcClient{
		runcPath: runcPath,
	}
}

func (*runcClient) CreateBundle(bundlePath string, jobSpec specs.Spec, user specs.User) error {
	err := os.MkdirAll(bundlePath, 0700)
	if err != nil {
		return err
	}

	rootfsPath := filepath.Join(bundlePath, "rootfs")
	err = os.MkdirAll(rootfsPath, 0700)
	if err != nil {
		return err
	}

	err = os.Chown(rootfsPath, int(user.UID), int(user.GID))
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(filepath.Join(bundlePath, "config.json"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		// This is super hard to test as we are root.
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	return enc.Encode(&jobSpec)
}

func (c *runcClient) RunContainer(pidFilePath, bundlePath, containerID string, stdout, stderr io.Writer) error {
	runcCmd := exec.Command(
		c.runcPath,
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

func (c *runcClient) ContainerState(containerID string) (specs.State, error) {
	runcCmd := exec.Command(
		c.runcPath,
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

func (c *runcClient) StopContainer(containerID string) error {
	runcCmd := exec.Command(
		c.runcPath,
		"kill",
		containerID,
	)

	return runcCmd.Run()
}

func (c *runcClient) DeleteContainer(containerID string) error {
	runcCmd := exec.Command(
		c.runcPath,
		"delete",
		"-f",
		containerID,
	)

	return runcCmd.Run()
}

func (*runcClient) DestroyBundle(bundlePath string) error {
	return os.RemoveAll(bundlePath)
}
