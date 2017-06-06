package commands

import (
	"crucible/config"
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
	RunE:  StartCommand,
}

func StartCommand(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		// fmt.Fprintln(os.Stderr, "must specify a job name")
		return errors.New("must specify a job name")
	}

	jobConfigPath := config.ConfigPath(args[0])
	_, err := config.ParseConfig(jobConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config at %s: %s", jobConfigPath, err.Error())
	}

	return nil
}
