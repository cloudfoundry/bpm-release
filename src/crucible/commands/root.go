package commands

import (
	"errors"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:           "crucible",
	Short:         "A bosh process manager for starting and stopping release jobs",
	Long:          "A bosh process manager for starting and stopping release jobs",
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("Exit code 1")
	},
}
