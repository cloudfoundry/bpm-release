package usertools

import (
	"errors"
	"os/user"
	"strconv"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

const VcapUser = "vcap"

//go:generate counterfeiter . UserFinder
type UserFinder interface {
	Lookup(username string) (specs.User, error)
}

type userFinder struct{}

func NewUserFinder() userFinder {
	return userFinder{}
}

func (f userFinder) Lookup(username string) (specs.User, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return specs.User{}, err
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return specs.User{}, err
	}
	if uid < 0 {
		return specs.User{}, errors.New("UID can't be negative")
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return specs.User{}, err
	}
	if gid < 0 {
		return specs.User{}, errors.New("GID can't be negative")
	}

	return specs.User{
		UID:      uint32(uid),
		GID:      uint32(gid),
		Username: u.Username,
	}, nil
}
