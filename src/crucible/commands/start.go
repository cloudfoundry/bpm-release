package commands

import (
	"crucible/config"
	"crucible/specbuilder"
	"errors"
	"fmt"
	"os/user"
	"strconv"

	specs "github.com/opencontainers/runtime-spec/specs-go"
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

	jobConfigPath := config.ConfigPath(args[0])
	jobConfig, err := config.ParseConfig(jobConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load config at %s: %s", jobConfigPath, err.Error())
	}

	_ = specbuilder.Build(args[0], jobConfig, idFinder{})

	return nil
}

type idFinder struct{}

// TODO: Test me
func (i idFinder) Lookup(username string) (specs.User, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return specs.User{}, err
	}

	// TODO: Can these be negative?
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return specs.User{}, err
	}

	// TODO: Can these be negative?
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return specs.User{}, err
	}

	return specs.User{
		UID:      uint32(uid),
		GID:      uint32(gid),
		Username: u.Username,
	}, nil
}
