package locksmith

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/lager"
	"golang.org/x/sys/unix"
)

type FileSystem struct {
	locksDir string
	lockType int
}

func NewExclusiveFileSystem(locksDir string) *FileSystem {
	return &FileSystem{
		locksDir: locksDir,
		lockType: unix.LOCK_EX,
	}
}

var FlockSyscall = unix.Flock

func (l *FileSystem) Lock(logger lager.Logger, key string) (*os.File, error) {
	if err := os.MkdirAll(l.locksDir, 0755); err != nil {
		return nil, err
	}
	key = strings.Replace(key, "/", "", -1)
	lockFile, err := os.OpenFile(l.path(key), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("creating lock file for key `%s`: %v", key, err)
	}

	fd := int(lockFile.Fd())
	logger.Info("acuiring-lock", lager.Data{"key": key, "lockfile": lockFile.Name()})
	if err := FlockSyscall(fd, l.lockType); err != nil {
		return nil, err
	}

	return lockFile, nil
}

func (l *FileSystem) Unlock(logger lager.Logger, lockFile *os.File) error {
	logger.Info("releasing-lock", lager.Data{"lockfile": lockFile.Name()})
	defer lockFile.Close()
	fd := int(lockFile.Fd())
	return FlockSyscall(fd, unix.LOCK_UN)
}

func (l *FileSystem) path(key string) string {
	return filepath.Join(l.locksDir, key+".lock")
}
