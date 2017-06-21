package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type CrucibleConfig struct {
	Name       string
	Executable string   `yaml:"executable"`
	Args       []string `yaml:"args"`
	Env        []string `yaml:"env"`
}

func ParseConfig(configPath string) (*CrucibleConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cconf := CrucibleConfig{}

	err = yaml.Unmarshal(data, &cconf)
	if err != nil {
		return nil, err
	}

	return &cconf, nil
}

func BoshRoot() string {
	boshRoot := os.Getenv("CRUCIBLE_BOSH_ROOT")
	if boshRoot == "" {
		boshRoot = "/var/vcap"
	}

	return boshRoot
}

func RuncPath() string {
	return filepath.Join(BoshRoot(), "packages", "runc", "bin", "runc")
}

func BundlesRoot() string {
	return filepath.Join(BoshRoot(), "data", "crucible", "bundles")
}
