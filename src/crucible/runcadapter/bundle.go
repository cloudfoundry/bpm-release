package runcadapter

import (
	"encoding/json"
	"os"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (a *runcAdapter) BuildBundle(bundlesRoot, jobName string, jobSpec specs.Spec) (string, error) {
	bundlePath := filepath.Join(bundlesRoot, jobName)
	err := os.MkdirAll(bundlePath, 0700)
	if err != nil {
		return "", err
	}
	rootfsPath := filepath.Join(bundlePath, "rootfs")
	err = os.MkdirAll(rootfsPath, 0700)
	if err != nil {
		return "", err
	}

	user, err := a.userIdFinder.Lookup("vcap") // hardcoded for now
	if err != nil {
		return "", err
	}

	err = os.Chown(rootfsPath, int(user.UID), int(user.GID))
	if err != nil {
		panic(err)
	}

	f, err := os.OpenFile(filepath.Join(bundlePath, "config.json"), os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		// This is super hard to test as we are root.
		return "", err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "\t")
	err = enc.Encode(&jobSpec)
	if err != nil {
		// Hard to test - spec was defined by golang so this should not be invalid json
		return "", err
	}

	return bundlePath, nil
}

func (a *runcAdapter) DestroyBundle(bundlesRoot, jobName string) error {
	return os.RemoveAll(filepath.Join(bundlesRoot, jobName))
}
