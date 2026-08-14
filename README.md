# `clic` is a CLI Composer

## Overview

`clic` composes reusable CLI trees without owning a process boundary.

* A `CLI` is a command line program entrypoint with at least one `Command`
* A `Command` is composed from `CommandMeta`, `CommandSpec`, `CommandState`, optional subcommands, and links
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

Commands that supply `Spec.Help` automatically accept `-h` and `--help`; the
requested command's help callback runs instead of its validator or handler.

## Option values

`StringOption` and `BoolOption` provide scalar values. `StringsOption`
accepts one delimited value from the command line or its environment variable:

```sh
tool --tag "one, two ,, three"
```

It uses a comma by default; set `OptionSpec.Separator` to use another literal
delimiter. Parsed values are trimmed and empty values are omitted, so the
example resolves to `[]string{"one", "two", "three"}`. Separators are not
escapable. String-slice defaults use native `[]string` values and receive the
same trimming and empty-value normalization.

Handlers can read effective values with `execution.Options.StringValue(key)`,
`BoolValue(key)`, and `StringValues(key)`. Each returns its zero value for
an absent key or a value of another kind.

## Presentation templates and remixing

`CommandMeta.Description`, `CommandMeta.Usage`, and `OptionMeta.Description`
are Go text templates. `DefaultHelp` supplies two values:

* `.CLI` is the root command name, such as `hyperctl`.
* `.Command` is the current command's full mounted path, such as
  `hyperctl theme render`.
* `.Name` is the literal name of the command or option being described.
  Option descriptions additionally receive `.ShortName` and `.EnvName`.

This lets a reusable command describe its own mounted invocation without a
host rewriting usage strings:

```go
Usage: []string{"{{ .Command }} --theme theme.json --data page.json --layout default"},
```

Names, aliases, and option spellings remain literal because clic needs them
before execution to validate and resolve the command tree. An empty
`CommandMeta.Name` or `OptionMeta.Name` defaults to its corresponding
specification key; set `Name` only when a host needs a different visible
spelling.
Custom help renderers can call `execution.MetadataTemplateData()` and
`meta.Render(...)` themselves.

`CommandMeta.EnvPrefix` scopes environment-name templates over a command tree.
An empty child prefix inherits its nearest ancestor prefix; a non-empty child
prefix overrides that value for its own declared options and descendants.
`OptionMeta.EnvName` templates receive only `.EnvPrefix`, so a reusable option
can declare `{{ .EnvPrefix }}CONFIG`. A prefix includes any delimiter it needs,
such as `HYPER_`.

Commands may also declare `Links`, which provide undocumented compatibility
paths to a descendant command without mounting a duplicate command tree:

```go
Links: clic.Links{
	{
		From:   "build",
		To: []string{"website", "build"},
	},
},
```

The link above makes `hyperctl build` execute the canonical `hyperctl website
build` command. Link destinations use normal command names or aliases and must
resolve to a descendant. A link cannot conflict with a mounted command name,
key, or alias, and default help does not list links.

See [`cmd/clippy`](./cmd/clippy) for a small executable example and
[`hsync/cli`](../hsync/cli) for an importable downstream adapter.
For agent guidance on implementing or remixing clic command trees, see the
[`clic CLI` skill](./skills/clic-cli/SKILL.md).

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
		Usage:       []string{"{{ .Command }} [--name name]"},
	}
}

func HelloSpec() clic.CommandSpec {
	name := HelloNameOption()
	return clic.CommandSpec{
		Key: "hello",
		Options: clic.MustOptionsFrom(func() clic.Option { return name }),
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

Use `clic.MustOptionsFrom` to compose options from factories. It rejects empty
and duplicate option keys when the command tree is constructed.

## List output

Commands can write a selected `ColumnListable` projection as an aligned table
or a JSON array. Each `clic.Field` uses its `Key` to obtain display values and
its `Header` for columnar output:

```go
err := clic.WriteList(execution.Stdout, clic.ColumnarFormat, skills,
	clic.Field{Key: "name", Header: "Name"},
	clic.Field{Key: "description", Header: "Description"},
)
```

Pass `clic.JSONFormat` to write the same selected fields as string-valued JSON
records. Use `clic.WriteJSON` when a command needs to write a typed JSON value
instead of a display-value projection.

## Roadmap

* Auto generate shell completion scripts
* Add a built-in `completion` subcommand for installing shell completion scripts
