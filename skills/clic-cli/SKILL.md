---
name: clic-cli
description: Implement or refactor Go command-line interfaces using github.com/herdworks/clic. Use for clic command factories, metadata, command remixing, environment prefixes, aliases, links, help, list output, and CLI tests.
---

# clic CLI

## Overview

Compose reusable command trees without putting process behavior in handlers.

## Workflow

1. Place command composition in a package-owned `<package>/cli` subpackage.
2. Define canonical commands and options, then mount or remix their factories.
3. Validate help, paths, environment names, and output at the shipped mount.

## Instructions

### Compose

- Keep `cmd/<binary>` thin: inject `clic.Execution` and call `CLI.Run`.
- Keep handlers free of process-global reads, framework-error printing, and exit.
- Define a canonical `var CLI`.
- Export fresh `Command()` or `Option()` factories only for reusable or
  remountable units.
- Prefer a single factory for each complete `clic.Command` or `clic.Option`.
  Do not split a one-off unit into separate `Meta()` and `Spec()` factories;
  extract either only when it is independently reused or is a meaningful shared
  unit.
- Put substantial leaf commands and their tests in separate files.
- Keep reusable behavior in `Spec` and host-facing presentation in `Meta`.
- Keep parsed values only in execution-local `State`.
- Use stable `Spec.Key` values as map keys.
- Omit `Meta.Name` when the key is the visible name.

### Mount and present

- Template command descriptions and usage with `.CLI`, `.Command`, and `.Name`.
- Template option descriptions with those values plus `.ShortName` and `.EnvName`.
- Template option environment names only with `.EnvPrefix`.
- Write reusable usage examples with `{{ .Command }}`.
- Set a root `EnvPrefix` and use `EnvName: "{{ .EnvPrefix }}NAME"`.
- Let an empty child prefix inherit; use a non-empty prefix to override a subtree.
- Clear a source prefix while remounting to inherit the host prefix; this does
  not disable environment names.
- Remix imported trees through `Command()` and change only the mounted root key
  or host metadata.
- Do not rewrite template-backed text or reconstruct imported children.

### Handle and output

- Use `clic.DefaultHelp` unless a custom renderer is necessary.
- Validate positional arguments in a command validator.
- Read resolved typed values from `Execution` in handlers.
- Implement `ColumnValues(field)` on the owning collection for list output.
- Call `clic.WriteList(execution.Stdout, format, rows, fields...)` for lists.
- Use `clic.WriteJSON` when JSON must retain typed values.

### Preserve compatibility and quality

- Use `CommandMeta.Aliases` for undocumented alternate names of one command.
- Use `Links` for undocumented deep compatibility paths to descendants.
- Use mounted names or aliases in `Link.To`.
- Fully expand clic composite literals one field per line, recursively.
- Test standalone factories and mounted help, paths, environment prefixes,
  aliases, links, and list formats.

## Tools

- Run `go fmt ./...` and `go test ./...` before handoff.

## References

- Consult the [clic README](../../README.md) for the complete API contract.
