package commands

import (
	"crucible/config"
	"crucible/runcadapter"
	"crucible/specbuilder"
	"errors"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(stopCommand)
}

var stopCommand = &cobra.Command{
	Use:   "stop <job-name>",
	Short: "Stops a BOSH Process",
	Long:  "Stops a BOSH Process",
	RunE:  stop,
}

func stop(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("must specify a job name")
	}

	jobName := args[0]

	adapter := runcadapter.NewRuncAdapater(config.RuncPath(), specbuilder.NewUserIDFinder())
	err := adapter.StopContainer(jobName)
	if err != nil {
		// test me?
		return err
	}

	return adapter.DestroyBundle(config.BundlesRoot(), jobName)
}
