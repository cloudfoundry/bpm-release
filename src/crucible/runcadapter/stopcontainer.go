package runcadapter

import "os/exec"

func (a *runcAdapter) StopContainer(jobName string) error {
	runcCmd := exec.Command(
		a.runcPath,
		"delete",
		"-f",
		jobName,
	)

	err := runcCmd.Start()
	if err != nil {
		return err
	}

	return runcCmd.Wait()
}
