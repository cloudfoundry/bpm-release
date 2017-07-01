package commands

import (
	"bpm/bpm"
	"errors"

	"code.cloudfoundry.org/lager"

	"github.com/spf13/cobra"
)

func init() {
	startCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	startCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the bpm configuration file.")
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

	return setupBpmLogs()
}

func start(cmd *cobra.Command, _ []string) error {
	cfg, err := bpm.ParseConfig(configPath)
	if err != nil {
		logger.Error("failed-to-parse-config", err)
		return err
	}

	logger = logger.Session("start", lager.Data{"process": cfg.Name})
	logger.Info("starting")
	defer logger.Info("complete")

	runcLifecycle := newRuncLifecycle()
	err = runcLifecycle.StartJob(jobName, cfg)
	if err != nil {
		logger.Error("failed-to-start", err)

		removeErr := runcLifecycle.RemoveJob(jobName, cfg)
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
