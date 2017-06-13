package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Long:          "A bosh process manager for starting and stopping release jobs",
	RunE:          root,
	Short:         "A bosh process manager for starting and stopping release jobs",
	SilenceErrors: true,
	Use:           "crucible",
	ValidArgs:     []string{"start", "stop"},
}

func root(cmd *cobra.Command, args []string) error {
	return errors.New("Exit code 1")
}
