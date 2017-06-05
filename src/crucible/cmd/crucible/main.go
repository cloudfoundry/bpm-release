package main

import (
	"crucible/config"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "must specify `start' or `stop'")
		usage()
	}

	fmt.Println(os.Args)

	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "must specify a job name")
		usage()
	}

	jobConfigPath := configPath(os.Args[2])
	_, err := config.ParseConfig(jobConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config at %s: %s", jobConfigPath, err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}

func usage() {
	fmt.Println("A bosh process manager for starting and stopping release jobs")
	fmt.Println("Usage: crucible [start|stop] <job name>")

	os.Exit(1)
}

func configPath(jobName string) string {
	configRoot := os.Getenv("CRUCIBLE_BOSH_ROOT")
	if configRoot == "" {
		configRoot = "/var/vcap"
	}

	return filepath.Join(configRoot, "jobs", jobName, "config", "crucible.yml")
}
