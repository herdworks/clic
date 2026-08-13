# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## Unreleased

## v0.3.2 (2026-08-12)

* Added `MustOptionsFrom` for composing option factories while rejecting empty
  and duplicate option keys.
* Added columnar and JSON list output for `ColumnListable` values through
  `WriteList`, plus `WriteJSON` for typed JSON values.

## v0.3.1 (2026-08-12)

* Added opt-in `-h` / `--help` parsing for commands that provide a `Help`
  callback. Help runs before required-option or command validation and instead
  of the command handler.
* Made `DefaultHelp` display non-empty effective option values when
  `ShowDefault` is enabled, and align long-only option names with short-form
  options.

## v0.3.0 (2026-08-12)

* **BREAKING:** Replaced the process-coupled mutable command framework with
  composable `CLI`, `Command`, and `Option` trees. Commands and options now
  split host metadata, reusable specifications, and execution-local state.
* Added injected `Execution` and `CLI.Run(execution, args) Result`; clic no
  longer reads process arguments, writes process streams, or exits.
* Added semantic command/option maps, command aliases, deterministic display
  ordering, scoped global options, string and bare boolean option parsing, and
  opt-in `DefaultHelp`.
* Added fresh command factories and `MustCommandsFrom` for reusable package
  adapters, with hsync as the first downstream composition.
* **BREAKING:** Renamed the module from `github.com/herdworks/clif` to
  `github.com/herdworks/clic`, and the root package from `clif` to `clic`.
* Repositioned clic as a CLI Composer.

## v0.2.0 (2026-08-11)

* Renamed the module from `github.com/herdworks/cli-go` to `github.com/herdworks/clif` 
* Renamed root package from `cli` to `clif`.
* Moved the Clippy example executable from `example/clippy.go` to `cmd/clippy`; 
  the module root is now importable by downstream programs.

## v0.1.0 (2025-08-31)

* Initial release of `github.com/herdworks/cli-go`, providing `CLI`, commands and subcommands, command options, argument parsing, validation, environment-backed defaults, and help output.
* Added the Clippy example command-line program.
