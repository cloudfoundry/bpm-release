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
	RootCmd.AddCommand(startCommand)
}

var startCommand = &cobra.Command{
	Long:  "Starts a BOSH Process",
	RunE:  start,
	Short: "Starts a BOSH Process",
	Use:   "start <job-name>",
}

func start(cmd *cobra.Command, args []string) error {
	if err := validateStartArguments(args); err != nil {
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

// Validate that a job name is provided.
// Not validating extra arguments is consitent with
// other CLI behavior
func validateStartArguments(args []string) error {
	if len(args) < 1 {
		return errors.New("must specify a job name")
	}

	return nil
}
