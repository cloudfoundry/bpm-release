package config

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

type BpmConfig struct {
	Name       string
	Executable string   `yaml:"executable"`
	Args       []string `yaml:"args"`
	Env        []string `yaml:"env"`
	Limits     *Limits  `yaml:"limits"`
}

type Limits struct {
	Memory    *string `yaml:"memory"`
	OpenFiles *uint64 `yaml:"open_files"`
	Processes *uint64 `yaml:"processes"`
}

func ParseConfig(configPath string) (*BpmConfig, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := BpmConfig{}

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	err = cfg.Validate()
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *BpmConfig) Validate() error {
	if c.Name == "" {
		return errors.New("invalid config: name")
	}

	if c.Executable == "" {
		return errors.New("invalid config: executable")
	}
	return nil
}

func BoshRoot() string {
	boshRoot := os.Getenv("BPM_BOSH_ROOT")
	if boshRoot == "" {
		boshRoot = "/var/vcap"
	}

	return boshRoot
}

func RuncPath() string {
	return filepath.Join(BoshRoot(), "packages", "runc", "bin", "runc")
}

func BundlesRoot() string {
	return filepath.Join(BoshRoot(), "data", "bpm", "bundles")
}

func RuncRoot() string {
	return filepath.Join(BoshRoot(), "data", "bpm", "runc")
}
