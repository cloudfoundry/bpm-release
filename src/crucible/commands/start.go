package commands

import (
	"crucible/config"
	"crucible/runcadapter"
	"crucible/specbuilder"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(startCommand)
}

var startCommand = &cobra.Command{
	Use:   "start",
	Short: "Starts a BOSH Process",
	Long:  "Starts a BOSH Process",
	RunE:  start,
}

func start(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("must specify a job name")
	}

	jobName := args[0]
	jobConfigPath := config.ConfigPath(jobName)
	jobConfig, err := config.ParseConfig(jobConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config at %s: %s", jobConfigPath, err.Error())
	}

	spec, err := specbuilder.Build(jobName, jobConfig, specbuilder.NewUserIDFinder())
	if err != nil {
		return fmt.Errorf("failed to load config at %s: %s", jobConfigPath, err.Error())
	}

	adapter := runcadapter.NewRuncAdapater(config.RuncPath())
	bundlePath, err := adapter.BuildBundle(config.BundlesRoot(), jobName, spec)
	if err != nil {
		return fmt.Errorf("bundle build failure: %s", err.Error())
	}

	return adapter.RunContainer(bundlePath, jobName)
}
