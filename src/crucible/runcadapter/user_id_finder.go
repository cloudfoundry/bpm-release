package runcadapter

import (
	"errors"
	"os/user"
	"strconv"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

//go:generate counterfeiter . UserIDFinder
type UserIDFinder interface {
	Lookup(username string) (specs.User, error)
}

type userIDFinder struct{}

func NewUserIDFinder() userIDFinder {
	return userIDFinder{}
}

func (f userIDFinder) Lookup(username string) (specs.User, error) {
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
