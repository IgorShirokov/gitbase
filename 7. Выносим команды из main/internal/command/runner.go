package command

import (
	"fmt"
	"io"
)

const (
	ExitSuccess      = 0
	ExitCommandError = 1
	ExitUsageError   = 2
)

type handler func(args []string, out io.Writer) error

type Runner struct {
	WorkingDirectory string
}

func NewRunner(workingDirectory string) *Runner {
	return &Runner{WorkingDirectory: workingDirectory}
}

func (r *Runner) Run(args []string, output, errorOutput io.Writer) int {
	if len(args) == 0 {
		printUsage(errorOutput)
		return ExitUsageError
	}

	registry := map[string]handler{
		"init": r.init,
		"add":  r.add,
	}

	command, found := registry[args[0]]
	if !found {
		fmt.Fprintf(errorOutput, "unknown error %q\n", args[0])
		printUsage(errorOutput)
		return ExitUsageError
	}

	if err := command(args[1:], output); err != nil {
		return ExitCommandError
	}

	return ExitSuccess
}

func (r *Runner) init(args []string, output io.Writer) error {
	if len(args) != 0 {
		return fmt.Errorf("init does not take arguments")
	}

	if err := initRepository(r.WorkingDirectory); err != nil {
		return err
	}

	fmt.Fprintln(output, ".gitbase initialized")

	return nil
}

func (r *Runner) add(args []string, output io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("Please specify at least one file to add")
	}

	if err := addFiles(r.WorkingDirectory, r.WorkingDirectory, args); err != nil {
		return err
	}

	fmt.Fprintln(output, "files added")

	return nil
}

func printUsage(output io.Writer) {
	fmt.Fprintln(output, "use: gitbase <command>")
	fmt.Fprintln(output, "commands: init, add")
}
