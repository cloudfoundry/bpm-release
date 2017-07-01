package main

import (
	"bpm/commands"
	"fmt"
	"os"
)

func main() {
	err := commands.RootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}
