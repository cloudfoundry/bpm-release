package commands

import (
	"bpm/config"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/spf13/cobra"
)

const DefaultStopTimeout = 20 * time.Second

func init() {
	stopCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	stopCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the bpm configuration file.")
	RootCmd.AddCommand(stopCommand)
}

var stopCommand = &cobra.Command{
	Long:    "Stops a BOSH Process",
	RunE:    stop,
	Short:   "Stops a BOSH Process",
	Use:     "stop <job-name>",
	PreRunE: stopPre,
}

func stopPre(cmd *cobra.Command, _ []string) error {
	if err := validateStopFlags(jobName, configPath); err != nil {
		return err
	}

	return setupBpmLogs()
}

func stop(cmd *cobra.Command, _ []string) error {
	jobConfig, err := config.ParseConfig(configPath)
	if err != nil {
		logger.Error("failed-to-parse-config", err)
		return err
	}

	logger = logger.Session("stop", lager.Data{"process": jobConfig.Name})
	logger.Info("starting")
	defer logger.Info("complete")

	runcLifecycle := newRuncLifecycle()
	err = runcLifecycle.StopJob(logger, jobName, jobConfig, DefaultStopTimeout)
	if err != nil {
		logger.Error("failed-to-stop", err)
	}

	return runcLifecycle.RemoveJob(jobName, jobConfig)
}

func validateStopFlags(jobName, configPath string) error {
	if jobName == "" {
		return errors.New("must specify a job")
	}

	if configPath == "" {
		return errors.New("must specify a configuration file")
	}

	return nil
}
