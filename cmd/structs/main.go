package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/rumpl/structs/internal/cli"
)

func run(args []string, stderr io.Writer) error {
	command, err := cli.NewCommand()
	if err == nil {
		command.SetArgs(args)
		err = command.Execute()
	}

	if err == nil {
		return nil
	}

	if _, errWrite := fmt.Fprintln(stderr, "Error:", err); errWrite != nil {
		return errors.Join(err, fmt.Errorf("write diagnostic: %w", errWrite))
	}

	return err
}

func main() {
	// This is the process boundary: diagnostics have been attempted and any
	// failure (including stderr failure) must produce a nonzero exit status.
	if err := run(os.Args[1:], os.Stderr); err != nil {
		os.Exit(1)
	}
}
