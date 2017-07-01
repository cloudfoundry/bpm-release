package commands

import (
	"bpm/bpm"
	"bpm/runc/adapter"
	"bpm/runc/client"
	"bpm/runc/lifecycle"
	"bpm/usertools"
	"errors"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"

	"github.com/spf13/cobra"
)

var jobName, configPath string
var logger lager.Logger
var userFinder = usertools.NewUserFinder()

var RootCmd = &cobra.Command{
	Long:          "A bosh process manager for starting and stopping release jobs",
	RunE:          root,
	Short:         "A bosh process manager for starting and stopping release jobs",
	SilenceErrors: true,
	Use:           "bpm",
	ValidArgs:     []string{"start", "stop", "list"},
}

func root(cmd *cobra.Command, args []string) error {
	return errors.New("Exit code 1")
}

func setupBpmLogs() error {
	bpmLogFileLocation := filepath.Join(bpm.BoshRoot(), "sys", "log", jobName, "bpm.log")
	err := os.MkdirAll(filepath.Join(bpm.BoshRoot(), "sys", "log", jobName), 0750)
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(bpmLogFileLocation, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0700)
	if err != nil {
		return err
	}

	usr, err := userFinder.Lookup(usertools.VcapUser)
	if err != nil {
		return err
	}

	err = os.Chown(bpmLogFileLocation, int(usr.UID), int(usr.GID))
	if err != nil {
		return err
	}

	logger, _ = lagerflags.NewFromConfig("bpm", lagerflags.DefaultLagerConfig())
	logger.RegisterSink(lager.NewWriterSink(logFile, lager.INFO))
	logger = logger.WithData(lager.Data{"job": jobName})

	return nil
}

func newRuncLifecycle() *lifecycle.RuncLifecycle {
	runcClient := client.NewRuncClient(bpm.RuncPath(), bpm.RuncRoot())
	runcAdapter := adapter.NewRuncAdapter()
	clock := clock.NewClock()

	return lifecycle.NewRuncLifecycle(
		runcClient,
		runcAdapter,
		userFinder,
		clock,
		bpm.BoshRoot(),
	)
}
