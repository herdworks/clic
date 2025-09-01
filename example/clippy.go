package main

import (
	"fmt"
	"strings"
)

var ClippyVersion = "0.1.0"
var Clippy = &CLI{
	Name:        "clippy",
	Version:     ClippyVersion,
	Description: "clippy is an example CLI tool.",
	Commands: []*Command{
		HelloCommand,
		VersionCommand,
	},
	Options: CommandOptions{
		{
			Name:        "help",
			ShortName:   "h",
			Description: "Display help information.",
			Boolean:     true,
			Secret:      true, // don't display a default value for this option
			Global:      true, // pass this option to all subcommands
		},
	},
}

var HelloCommand = &Command{
	Config: CommandConfig{
		Name:        "hello",
		Description: "Say hello",
		Usage:       "clippy hello <name> [options]",
		Validator:   HelloCommandValidator,
		Handler:     HelloCommandHandler,
	},
	Options: CommandOptions{
		{
			Name:        "title",
			ShortName:   "t",
			EnvName:     "CLIPPY_TITLE",
			Description: "Title to use in the greeting",
			Required:    false,
			Secret:      false,
		},
	},
}

func HelloCommandValidator(cmd *Command) error {
	requiredArgs := 1
	if len(cmd.Args) != requiredArgs {
		return fmt.Errorf("expected %v argument(s), got %v (%s)", requiredArgs, len(cmd.Args), strings.Join(cmd.Args, ", "))
	}
	return nil
}

func HelloCommandHandler(cmd *Command) (err error) {
	var title string = cmd.Options.Get("title")
	var name string = cmd.Args[0]
	if len(title) > 0 {
		fmt.Printf("Hello, %s %s!\n", title, name)
	} else {
		fmt.Printf("Hello, %s!\n", name)
	}
	return nil
}

var VersionCommand = &Command{
	Config: CommandConfig{
		Name:        "version",
		Description: "Display version information",
		Usage:       "clippy version",
		Validator:   nil,
		Handler:     VersionCommandHandler,
	},
	Options: CommandOptions{},
}

func VersionCommandHandler(cmd *Command) (err error) {
	fmt.Printf("clippy version %s\n", ClippyVersion)
	return nil
}

func main() {
	Clippy.Execute()
}
