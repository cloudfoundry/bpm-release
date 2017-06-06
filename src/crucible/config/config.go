package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

type CrucibleConfig struct {
	Process *Process `yaml:"process"`
}

type Process struct {
	Name       string   `yaml:"name"`
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

func ConfigPath(jobName string) string {
	configRoot := os.Getenv("CRUCIBLE_BOSH_ROOT")
	if configRoot == "" {
		configRoot = "/var/vcap"
	}

	return filepath.Join(configRoot, "jobs", jobName, "config", "crucible.yml")
}
