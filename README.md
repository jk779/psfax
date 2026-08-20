# psfax

`psfax` is a small macOS command-line tool that provides a process tree similar to Linux `ps fax`, using macOS's native `/bin/ps` as its data source.

It is intentionally macOS-only. The universal release binary supports both Apple Silicon (`arm64`) and Intel (`amd64`) Macs.

## Features

- Process tree output with PID, CPU, memory, user, and command columns.
- Filters for a PID subtree, user, or command substring.
- Terminal-aware command truncation and optional executable highlighting.
- No third-party runtime dependencies.

## Requirements

- macOS.
- Go 1.27.0 for development and local builds.
- `asdf` with the version from `.tool-versions`, or an equivalent Go 1.27.0 installation.
- Xcode Command Line Tools for `lipo` and `strip` when building a universal binary.

## Usage

```text
psfax
psfax --user alice
psfax --sub iterm
psfax --pid 1234
psfax --wide
```

Run `psfax --help` for the complete option list.

## Development

```bash
make check       # format check, tests, and go vet
make build       # build dist/psfax for the release targets
make package     # create a versioned universal archive in dist/
make clean       # remove dist/
```

The Makefile deliberately keeps generated binaries and archives under `dist/`; they are not committed to the repository.

## License

Copyright (C) 2026 Michael.

This project is licensed under the GNU Affero General Public License, version 3.0 or later. See [LICENSE](LICENSE) and the [official license text](https://www.gnu.org/licenses/agpl-3.0.html).
