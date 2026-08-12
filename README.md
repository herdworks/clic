# `clif` is a CLI Framework

## Overview

* A [`CLI`](./cli.go) is a program with one or more Commands.

* A [`Command`](./cli.go) is an action that can be performed by the program.

* A [`CommandConfig`](./cli.go) is used to configure a Command.

  * `Name` configures the command or subcommand name
  * `Description` configures the command description in `--help` output
  * `Usage` configures the usage example(s) in `--help` output
  * `Validator` configures the command validation function
  * `Handler` configures the command processing function

* A [`CommandOption`](./cli.go) is used to configure a Command option (e.g. a flag).

  * `Name` and `ShortName` configure long and short names (e.g. `--example` and `-e`)
  * `Description` configures the option description in `--help` output
  * `EnvName` configures an environment variable name to override the default option value with
  * `Boolean` configures a boolean option (i.e. set `Boolean: true` to add an option that takes no arguments)
  * `Default` configures the default option value
  * `Required` configures a required option, with auto-validation
  * `Global` configures subcommand inheritance for a given option
  * `Secret` configures option value redaction (hides display of default values in `--help` output)

* Built-in helper utilities, including basic columnar layouts.

See [`cmd/clippy`](./cmd/clippy) for a minimal external consumer that registers a global `--help` flag and runs `Clippy.Execute()`.

## Example

Build the example CLI program (`clippy`) using `go build`:

```shell
go build -o clippy ./cmd/clippy
```

## Roadmap

* Auto generate shell completion scripts
* Add a built-in `completion` subcommand for installing shell completion scripts
