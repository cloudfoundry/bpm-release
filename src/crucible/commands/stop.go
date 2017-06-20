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
	RootCmd.AddCommand(stopCommand)
}

var stopCommand = &cobra.Command{
	Long:  "Stops a BOSH Process",
	RunE:  stop,
	Short: "Stops a BOSH Process",
	Use:   "stop <job-name>",
}

func stop(cmd *cobra.Command, args []string) error {
	if err := validateStopArguments(args); err != nil {
		return err
	}

	jobName := args[0]
	jobConfigPath := config.ConfigPath(jobName)
	jobConfig, err := config.ParseConfig(jobConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config at %s: %s", jobConfigPath, err.Error())
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

// Validate that a job name is provided.
// Not validating extra arguments is consitent with
// other CLI behavior
func validateStopArguments(args []string) error {
	if len(args) < 1 {
		return errors.New("must specify a job name")
	}

	return nil
}
