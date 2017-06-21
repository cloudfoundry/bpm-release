package commands

import (
	"crucible/config"
	"crucible/runcadapter"
	"errors"
	"fmt"

	"code.cloudfoundry.org/clock"

	"github.com/spf13/cobra"
)

func init() {
	startCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	startCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the crucible configuration file.")
	RootCmd.AddCommand(startCommand)
}

var startCommand = &cobra.Command{
	Long:  "Starts a BOSH Process",
	RunE:  start,
	Short: "Starts a BOSH Process",
	Use:   "start <job-name>",
}

func start(cmd *cobra.Command, _ []string) error {
	if err := validateStartFlags(jobName, configPath); err != nil {
		return err
	}

	jobConfig, err := config.ParseConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config at %s: %s", configPath, err.Error())
	}

	userIDFinder := runcadapter.NewUserIDFinder()
	runcAdapter := runcadapter.NewRuncAdapter(config.RuncPath(), userIDFinder)
	clock := clock.NewClock()

	jobLifecycle := runcadapter.NewRuncJobLifecycle(
		runcAdapter,
		clock,
		jobName,
		jobConfig,
	)

	err = jobLifecycle.StartJob()
	if err != nil {
		removeErr := jobLifecycle.RemoveJob()
		if removeErr != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "failed to remove failed job: %s\n", removeErr.Error())
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
