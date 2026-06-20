# Architecture

## Entry points

- `main.go` defines CLI flags, environment binding, command parsing, defaults, and usage output.
- `internal/entry` converts parsed commands into runtime calls and handles daemon, signal, and logging setup.

## Runtime packages

- `internal/copier` coordinates scanning, chunking, joining, retry-aware execution, and result aggregation.
- `internal/copier/find` provides file discovery using external `find` or Go scanning paths.
- `internal/copier/native` implements native file copying.
- `internal/copier/rsync` builds and runs rsync tasks.
- `internal/syncer` handles mount/sandbox preparation and invokes the copier.
- `internal/selector` parses copy-entry CSV data and fans out sync child processes.

## Shared support

- `internal/args` centralizes typed access to Viper-backed configuration.
- `internal/returns` defines serializable result and file/mount information types.
- `internal/sys` wraps platform-specific filesystem, mount, fd, and process helpers.
- `internal/util` contains logging, lookup, conversion, unit parsing, and flag binding helpers.

## Boundaries

The CLI and docs should treat `main.go` plus `internal/entry` as the command contract. Lower-level packages should remain focused on data movement and process orchestration; avoid mixing selector daemon concerns into copier internals.
