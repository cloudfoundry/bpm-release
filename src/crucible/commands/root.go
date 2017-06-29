package commands

import (
	"crucible/config"
	"crucible/runcadapter"
	"errors"
	"os"
	"path/filepath"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"

	"github.com/spf13/cobra"
)

var jobName, configPath string
var logger lager.Logger

var RootCmd = &cobra.Command{
	Long:          "A bosh process manager for starting and stopping release jobs",
	RunE:          root,
	Short:         "A bosh process manager for starting and stopping release jobs",
	SilenceErrors: true,
	Use:           "crucible",
	ValidArgs:     []string{"start", "stop"},
}

func root(cmd *cobra.Command, args []string) error {
	return errors.New("Exit code 1")
}

func setupCrucibleLogs(cmd *cobra.Command, args []string) error {
	crucibleLogFileLocation := filepath.Join(config.BoshRoot(), "sys", "log", jobName, "crucible.log")
	err := os.MkdirAll(filepath.Join(config.BoshRoot(), "sys", "log", jobName), 0750)
	if err != nil {
		return err
	}

	logFile, err := os.OpenFile(crucibleLogFileLocation, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0700)
	if err != nil {
		return err
	}

	usr, err := runcadapter.NewUserIDFinder().Lookup(runcadapter.VCAP_USER)
	if err != nil {
		return err
	}

	err = os.Chown(crucibleLogFileLocation, int(usr.UID), int(usr.GID))
	if err != nil {
		return err
	}

	logger, _ = lagerflags.NewFromConfig("crucible", lagerflags.DefaultLagerConfig())
	logger.RegisterSink(lager.NewWriterSink(logFile, lager.INFO))
	logger = logger.WithData(lager.Data{"job": jobName})
	return nil
}
