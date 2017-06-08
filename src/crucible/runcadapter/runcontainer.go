package runcadapter

import (
	"fmt"
	"os/exec"
	"time"
)

func (a *runcAdapter) RunContainer(bundlePath, jobName string) error {
	runcCmd := exec.Command(a.runcPath, "run", "--bundle", bundlePath, "--detach", jobName)

	err := runcCmd.Start()
	if err != nil {
		return err
	}

	return poll(a.runcPath, jobName)
}

/*
*  Polling is necessary as the cmd functions do not track
*  the runc --detach correctly.
 */
func poll(runcPath, jobName string) error {
	timeout := 5 * time.Second
	timeoutChan := time.After(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)

	for {
		select {
		case <-timeoutChan:
			ticker.Stop()
			return fmt.Errorf("timed out after %v", timeout)
		case <-ticker.C:
			_, err := exec.Command(runcPath, "state", jobName).CombinedOutput()
			if err == nil { // intentional logic
				return nil
			}
		}
	}
}
