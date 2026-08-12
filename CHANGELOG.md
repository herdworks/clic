# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## Unreleased

## v0.2.0 (2026-08-11)

* Renamed the module from `github.com/herdworks/cli-go` to `github.com/herdworks/clif` 
* Renamed root package from `cli` to `clif`.
* Moved the Clippy example executable from `example/clippy.go` to `cmd/clippy`; 
  the module root is now importable by downstream programs.

## v0.1.0 (2025-08-31)

* Initial release of `github.com/herdworks/cli-go`, providing `CLI`, commands and subcommands, command options, argument parsing, validation, environment-backed defaults, and help output.
* Added the Clippy example command-line program.
