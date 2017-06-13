package commands

import (
	"crucible/config"
	"crucible/lifecycle"
	"crucible/specbuilder"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

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

	userIDFinder := specbuilder.NewUserIDFinder()

	jobLifecycle := lifecycle.NewRuncJobLifecycle(config.RuncPath(),
		config.BundlesRoot(),
		jobName,
		jobConfig,
		userIDFinder,
	)
	return jobLifecycle.StopJob()
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
