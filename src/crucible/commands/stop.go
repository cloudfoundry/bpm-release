package commands

import (
	"crucible/config"
	"crucible/runcadapter"
	"errors"
	"fmt"
	"time"

	"code.cloudfoundry.org/clock"

	"github.com/spf13/cobra"
)

const DEFAULT_STOP_TIMEOUT = 20 * time.Second

func init() {
	stopCommand.Flags().StringVarP(&jobName, "job", "j", "", "The job name.")
	stopCommand.Flags().StringVarP(&configPath, "config", "c", "", "The path to the crucible configuration file.")
	RootCmd.AddCommand(stopCommand)
}

var stopCommand = &cobra.Command{
	Long:  "Stops a BOSH Process",
	RunE:  stop,
	Short: "Stops a BOSH Process",
	Use:   "stop <job-name>",
}

func stop(cmd *cobra.Command, _ []string) error {
	if err := validateStopFlags(jobName, configPath); err != nil {
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

	err = jobLifecycle.StopJob(DEFAULT_STOP_TIMEOUT)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "failed to stop job: %s\n", err.Error())
	}

	return jobLifecycle.RemoveJob()
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
