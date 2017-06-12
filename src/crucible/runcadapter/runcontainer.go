package runcadapter

import "os/exec"

func (a *runcAdapter) RunContainer(bundlePath, jobName string) error {
	runcCmd := exec.Command(a.runcPath, "run", "--bundle", bundlePath, "--detach", jobName)

	err := runcCmd.Start()
	if err != nil {
		return err
	}

	return runcCmd.Wait()
}
