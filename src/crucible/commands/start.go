package commands

import (
	"crucible/config"
	"crucible/runcadapter"
	"errors"

	"code.cloudfoundry.org/clock"
	"code.cloudfoundry.org/lager"

	"github.com/spf13/cobra"
)

func init() {
	startCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	startCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the crucible configuration file.")
	RootCmd.AddCommand(startCommand)
}

var startCommand = &cobra.Command{
	Long:              "Starts a BOSH Process",
	RunE:              start,
	Short:             "Starts a BOSH Process",
	Use:               "start <job-name>",
	PersistentPreRunE: startPre,
}

func startPre(cmd *cobra.Command, _ []string) error {
	if err := validateStartFlags(jobName, configPath); err != nil {
		return err
	}

	return setupCrucibleLogs(cmd, []string{})
}

func start(cmd *cobra.Command, _ []string) error {
	jobConfig, err := config.ParseConfig(configPath)
	if err != nil {
		logger.Error("failed-to-parse-config", err)
		return err
	}

	logger = logger.Session("start", lager.Data{"process": jobConfig.Name})
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

	err = jobLifecycle.StartJob(jobName, jobConfig)
	if err != nil {
		logger.Error("failed-to-start", err)

		removeErr := jobLifecycle.RemoveJob(jobName, jobConfig)
		if removeErr != nil {
			logger.Error("failed-to-cleanup", removeErr)
		}

		return err
	}

	return nil
}

func validateStartFlags(jobName, configPath string) error {
	if jobName == "" {
		return errors.New("must specify a job")
	}

	if configPath == "" {
		return errors.New("must specify a configuration file")
	}

	return nil
}
