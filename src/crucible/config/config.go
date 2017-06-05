package config

import "os"

type CrucibleConfig struct{}

func ParseConfig(configPath string) (*CrucibleConfig, error) {
	_, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
