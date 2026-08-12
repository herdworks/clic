// Command clippy demonstrates an application built with clic.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/herdworks/clic"
)

const clippyVersion = "0.3.0"

func main() {
	execution := clic.Execution{
		Context:          context.Background(),
		Stdin:            os.Stdin,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
		LookupEnv:        os.LookupEnv,
		WorkingDirectory: ".",
	}
	result := NewCLI().Run(execution, os.Args[1:])
	if result.Error != nil {
		fmt.Fprintln(os.Stderr, result.Error)
	}
	os.Exit(result.ExitCode)
}

func NewCLI() clic.CLI {
	return clic.CLI{
		Command: clic.Command{
			Meta: clic.CommandMeta{Name: "clippy", Description: "clippy is an example CLI tool."},
			Spec: clic.CommandSpec{Key: "clippy", Help: help},
			Commands: clic.Commands{
				"hello":   NewHelloCommand(),
				"version": NewVersionCommand(),
			},
		},
		Version: clippyVersion,
	}
}

func NewHelloCommand() *clic.Command {
	return &clic.Command{
		Meta: clic.CommandMeta{Name: "hello", Description: "Say hello", Usage: []string{"clippy hello <name> [options]"}},
		Spec: clic.CommandSpec{
			Key: "hello",
			Options: clic.Options{
				"title": {Meta: clic.OptionMeta{Name: "title", ShortName: "t", EnvName: "CLIPPY_TITLE", Description: "Title to use in the greeting"}, Spec: clic.OptionSpec{Key: "title", Kind: clic.StringOption}},
			},
			Validator: helloValidator,
			Handler:   hello,
		},
	}
}

func NewVersionCommand() *clic.Command {
	return &clic.Command{
		Meta: clic.CommandMeta{Name: "version", Description: "Display version information", Usage: []string{"clippy version"}},
		Spec: clic.CommandSpec{Key: "version", Handler: version},
	}
}

func helloValidator(execution *clic.Execution) error {
	if len(execution.Command.State.Arguments) != 1 {
		return fmt.Errorf("expected 1 argument, got %d (%s)", len(execution.Command.State.Arguments), strings.Join(execution.Command.State.Arguments, ", "))
	}
	return nil
}

func hello(execution *clic.Execution) error {
	title := execution.Options["title"].StringValue()
	name := execution.Command.State.Arguments[0]
	if title != "" {
		_, err := fmt.Fprintf(execution.Stdout, "Hello, %s %s!\n", title, name)
		return err
	}
	_, err := fmt.Fprintf(execution.Stdout, "Hello, %s!\n", name)
	return err
}

func version(execution *clic.Execution) error {
	_, err := fmt.Fprintf(execution.Stdout, "clippy version %s\n", clippyVersion)
	return err
}

func help(meta clic.CommandMeta, execution *clic.Execution) error {
	_, err := fmt.Fprintf(execution.Stdout, "%s\n", meta.Description)
	return err
}
