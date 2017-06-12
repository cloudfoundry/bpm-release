package runcadapter

import (
	"crucible/config"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func (a *runcAdapter) RunContainer(bundlePath, jobName string) error {
	cruciblePidDir := filepath.Join(config.BoshRoot(), "sys", "run", "crucible")
	err := os.MkdirAll(cruciblePidDir, 0700)
	if err != nil {
		// Test Me
		return err
	}

	runcCmd := exec.Command(
		a.runcPath,
		"run",
		"--bundle", bundlePath,
		"--pid-file", filepath.Join(cruciblePidDir, fmt.Sprintf("%s.pid", jobName)),
		"--detach",
		jobName,
	)

	err = runcCmd.Start()
	if err != nil {
		return err
	}

	return runcCmd.Wait()
}
