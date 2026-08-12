# `clic` is a CLI Composer

## Overview

`clic` composes reusable CLI trees without owning a process boundary.

* A `CLI` is a command line program entrypoint with at least one `Command`
* A `Command` is composed from `CommandMeta`, `CommandSpec`, `CommandState`, and optional subcommands
* `CommandMeta` defines command metadata (name, description, usage, etc)
* `CommandSpec` defines command options plus help, validation, and handler functions
* An `Option` is composed from `OptionMeta`, `OptionSpec`, and `OptionState`
* `OptionMeta` defines the option name, shortname, description, environment variable, default value and more
* `OptionSpec` defines the underlying option type and required status
* `OptionState` holds parsed options and their values
* An `Execution` is a single unit of work performed by a CLI

## Composability

A `CLI` embeds an entrypoint `Command` and adds program-level metadata such as `Version`.
Commands and Options divide ownership three ways:

* `Meta` is all presentation.
  Command and Option names, aliases, descriptions, usage, defaults, and display order.
  A host CLI can remix this metadata without changing what a command does.
* `Spec` is reusable implementation.
  Command and Option keys, kinds, requiredness, help output, validation, and handler functions.
  Commands and Options are stored in maps that are traversable by their respective keys.
* `State` is runtime data.
  Commands and Options use state to store user-inputs during execution.

An `Execution` is one call to `CLI.Run`.
It supplies runtime dependencies; `Run` resolves a command, its effective
options, and user input before calling its handler or help. It returns a
[`Result`](./cli.go) without reading process globals or exiting.

See [`cmd/clippy`](./cmd/clippy) for a small executable example and
[`hsync/cli`](../hsync/cli) for an importable downstream adapter.

## Example

Build the example CLI program (`clippy`) using `go build`:

```shell
go build -o clippy ./cmd/clippy
```

## Usage

Keep the executable thin: it injects process dependencies and runs the sibling `cli` package's canonical composition.

`<package>/cmd/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/example/package/cli"
	"github.com/herdworks/clic"
)

func main() {
	execution := clic.Execution{
		Context:          context.Background(),
		Stdin:            os.Stdin,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
		LookupEnv:        os.LookupEnv,
		WorkingDirectory: ".",
	}
	result := cli.CLI.Run(execution, os.Args[1:])
	if result.Error != nil {
		fmt.Fprintln(os.Stderr, result.Error)
	}
	os.Exit(result.ExitCode)
}
```

`<package>/cli/cli.go` declares the composed CLI and collects fresh command factories:

```go
package cli

import "github.com/herdworks/clic"

var CLI = clic.CLI{
	Command: clic.Command{
		Meta: clic.CommandMeta{
			Name:        "example",
			Description: "An example program.",
		},
		Spec: clic.CommandSpec{
			Key:  "example",
			Help: clic.DefaultHelp,
		},
		Commands: clic.MustCommandsFrom(
			HelloCommand,
			// add additional commands factories go here
		),
	},
}
```

Each command should have its own file. Define the command, then its metadata
and specification, followed by its options and private behavior.

`<package>/cli/hello.go`:

```go
package cli

import (
	"fmt"

	"github.com/herdworks/clic"
)

// hello command
func HelloCommand() clic.Command {
	return clic.Command{
		Meta: HelloMeta(),
		Spec: HelloSpec(),
	}
}

func HelloMeta() clic.CommandMeta {
	return clic.CommandMeta{
		Name:        "hello",
		Description: "Print a greeting.",
		Usage:       []string{"example hello [--name name]"},
	}
}

func HelloSpec() clic.CommandSpec {
	name := HelloNameOption()
	return clic.CommandSpec{
		Key: "hello",
		Options: clic.Options{
			"name": &name,
		},
		Handler: helloHandler,
	}
}

// hello --name option
func HelloNameOption() clic.Option {
	return clic.Option{
		Meta: HelloNameMeta(),
		Spec: HelloNameSpec(),
	}
}

func HelloNameMeta() clic.OptionMeta {
	return clic.OptionMeta{
		Name:        "name",
		Description: "Name to greet.",
	}
}

func HelloNameSpec() clic.OptionSpec {
	return clic.OptionSpec{
		Key:  "name",
		Kind: clic.StringOption,
	}
}

// hello handler
func helloHandler(execution *clic.Execution) error {
	_, err := fmt.Fprintf(execution.Stdout, "Hello, %s!\\n", execution.Options["name"].StringValue())
	return err
}
```

## Roadmap

* Auto generate shell completion scripts
* Add a built-in `completion` subcommand for installing shell completion scripts
