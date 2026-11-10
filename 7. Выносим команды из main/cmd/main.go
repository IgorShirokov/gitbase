package main

import (
	"fmt"
	"gitbase/internal/command"
	"os"
)

func main() {
	workingDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(command.ExitCommandError)
	}

	runner := command.NewRunner(workingDirectory)
	os.Exit(runner.Run(os.Args[1:], os.Stdout, os.Stderr))
}
