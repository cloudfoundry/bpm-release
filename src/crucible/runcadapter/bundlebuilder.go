package runcadapter

import (
	"encoding/json"
	"os"
	"path/filepath"

	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func (a *runcAdapter) BuildBundle(bundleRoot, jobName string, jobSpec specs.Spec) (string, error) {
	bundlePath := filepath.Join(bundleRoot, jobName)
	err := os.MkdirAll(bundlePath, 0700)
	if err != nil {
		return "", err
	}

	err = os.MkdirAll(filepath.Join(bundlePath, "rootfs"), 0700)
	if err != nil {
		return "", err
	}

	f, err := os.OpenFile(filepath.Join(bundlePath, "config.json"), os.O_RDWR|os.O_CREATE, 0700)
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

	err = chownR(bundlePath, 2000, 3000) // TODO: user UserIDFinder
	if err != nil {
		panic(err)
	}

	return bundlePath, nil
}

func chownR(path string, uid, gid int) error {
	return filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if err == nil {
			err = os.Chown(name, uid, gid)
		}
		return err
	})
}
