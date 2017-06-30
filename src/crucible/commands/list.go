package commands

import (
	"crucible/config"
	"crucible/presenters"
	"crucible/runcadapter"
	"fmt"

	"code.cloudfoundry.org/clock"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(listCommandCommand)
}

var listCommandCommand = &cobra.Command{
	Long:  "Lists the state of crucible containers",
	RunE:  listContainers,
	Short: "List containers",
	Use:   "list",
}

func listContainers(cmd *cobra.Command, _ []string) error {
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

	jobs, err := jobLifecycle.ListJobs()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "failed to list jobs: %s\n", err.Error())
		return err
	}

	err = presenters.PrintJobs(jobs, cmd.OutOrStdout())
	if err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "failed to display jobs: %s\n", err.Error())
		return err
	}

	return nil
}
