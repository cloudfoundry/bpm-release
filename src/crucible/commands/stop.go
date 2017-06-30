package commands

import (
	"crucible/config"
	"crucible/runcadapter"
	"errors"
	"time"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"github.com/spf13/cobra"
)

const DEFAULT_STOP_TIMEOUT = 20 * time.Second

func init() {
	stopCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	stopCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the crucible configuration file.")
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

	return setupCrucibleLogs(cmd, []string{})
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

	runcClient := runcadapter.NewRuncClient(config.RuncPath(), config.RuncRoot())
	runcAdapter := runcadapter.NewRuncAdapter()
	userIDFinder := runcadapter.NewUserIDFinder()
	clock := clock.NewClock()

	jobLifecycle := runcadapter.NewRuncJobLifecycle(
		runcClient,
		runcAdapter,
		userIDFinder,
		clock,
		config.BoshRoot(),
	)

	err = jobLifecycle.StopJob(logger, jobName, jobConfig, DEFAULT_STOP_TIMEOUT)
	if err != nil {
		logger.Error("failed-to-stop", err)
	}

	return jobLifecycle.RemoveJob(jobName, jobConfig)
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
