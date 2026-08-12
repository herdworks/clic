// Command clippy demonstrates an application built with clif.
package main

import (
	"fmt"
	"strings"

	"github.com/herdworks/clif"
)

var ClippyVersion = "0.2.0"

var Clippy = &clif.CLI{
	Name:        "clippy",
	Version:     ClippyVersion,
	Description: "clippy is an example CLI tool.",
	Commands: clif.Commands{
		HelloCommand,
		VersionCommand,
	},
	Options: clif.CommandOptions{
		{
			Name:        "help",
			ShortName:   "h",
			Description: "Display help information.",
			Boolean:     true,
			Secret:      true,
			Global:      true,
		},
	},
}

var HelloCommand = &clif.Command{
	Config: clif.CommandConfig{
		Name:        "hello",
		Description: "Say hello",
		Usage:       "clippy hello <name> [options]",
		Validator:   HelloCommandValidator,
		Handler:     HelloCommandHandler,
	},
	Options: clif.CommandOptions{
		{
			Name:        "title",
			ShortName:   "t",
			EnvName:     "CLIPPY_TITLE",
			Description: "Title to use in the greeting",
		},
	},
}

var VersionCommand = &clif.Command{
	Config: clif.CommandConfig{
		Name:        "version",
		Description: "Display version information",
		Usage:       "clippy version",
		Handler:     VersionCommandHandler,
	},
}

func main() {
	Clippy.Execute()
}

func HelloCommandValidator(cmd *clif.Command) error {
	if len(cmd.Args) != 1 {
		return fmt.Errorf("expected 1 argument, got %d (%s)", len(cmd.Args), strings.Join(cmd.Args, ", "))
	}
	return nil
}

func HelloCommandHandler(cmd *clif.Command) error {
	title := cmd.Options.Get("title")
	name := cmd.Args[0]
	if title != "" {
		fmt.Printf("Hello, %s %s!\n", title, name)
		return nil
	}
	fmt.Printf("Hello, %s!\n", name)
	return nil
}

func VersionCommandHandler(*clif.Command) error {
	fmt.Printf("clippy version %s\n", ClippyVersion)
	return nil
}
