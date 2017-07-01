package commands

import (
	"bpm/presenters"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(listCommandCommand)
}

var listCommandCommand = &cobra.Command{
	Long:  "Lists the state of bpm containers",
	RunE:  listContainers,
	Short: "List containers",
	Use:   "list",
}

func listContainers(cmd *cobra.Command, _ []string) error {
	runcLifecycle := newRuncLifecycle()
	jobs, err := runcLifecycle.ListJobs()
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
