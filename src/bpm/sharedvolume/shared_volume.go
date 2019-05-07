package sharedvolume

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/lager"
	"github.com/opencontainers/runc/libcontainer/mount"
	"golang.org/x/sys/unix"
)

type Locksmith interface {
	Lock(logger lager.Logger, key string) (*os.File, error)
	Unlock(logger lager.Logger, lockFile *os.File) error
}

type Factory struct {
	locksmith Locksmith
}

func NewFactory(locksmith Locksmith) *Factory {
	return &Factory{locksmith: locksmith}
}

func (f *Factory) Create(logger lager.Logger, path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to mkdir %q: %v", path, err)
	}

	if err := f.ensureMounted(logger, path); err != nil {
		return err
	}

	if err := makeSharedMount(logger, path); err != nil {
		return err
	}

	return nil
}

func (f *Factory) ensureMounted(logger lager.Logger, path string) error {
	lock, err := f.locksmith.Lock(logger, path)
	if err != nil {
		return err
	}
	defer f.locksmith.Unlock(logger, lock)

	mounted, err := mount.Mounted(path)
	if err != nil {
		return fmt.Errorf("failed to check whether %q is already mounted: %v", path, err)
	}
	if mounted {
		logger.Info("already-mounted", lager.Data{"path": path})
		return nil
	}

	logger.Info("bind-mounting", lager.Data{"path": path})
	if err := unix.Mount(path, path, "none", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("failed to bind mount %q: %v", path, err)
	}

	return nil
}

func makeSharedMount(logger lager.Logger, path string) error {
	logger.Info("making-shared-mount", lager.Data{"path": path})
	if err := unix.Mount("", path, "", unix.MS_SHARED, ""); err != nil {
		return fmt.Errorf("failed to make mount %q shared: %v", path, err)
	}

	return nil
}
