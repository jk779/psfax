# Development Guide

## Scope

psfax is deliberately a macOS-only command-line tool. Keep the implementation based on macOS `/bin/ps` and preserve the universal `arm64`/`amd64` release build unless the project direction changes explicitly.

## Toolchain

Use the Go version selected by `.tool-versions`:

```bash
asdf exec go version
```

Do not install another runtime for normal development. Keep project artifacts, comments, and documentation in English.

## Validation

Before committing Go changes, run:

```bash
make check
make build
```

Keep tests focused on deterministic parsing, filtering, tree selection, and rendering helpers. Do not make unit tests depend on the live process table.

## Changes

Keep the executable-highlighting heuristics conservative. Changes to command detection should include focused tests for bundle paths, ordinary executable paths, arguments, and Unicode command lines.

Generated binaries belong in `dist/` and should not be committed. Releases publish the universal `dist/psfax` binary directly; do not reintroduce archive packaging unless the release design changes explicitly.
