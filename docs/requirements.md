# Requirements

`fast-volume-syncer` must provide safe, repeatable copy and synchronization workflows for storage migration or replication jobs.

## Functional scope

- `copy SRC_PATH DST_PATH` copies one source path tree into one destination path tree.
- `sync SRC_PATH [SRC_SUBPATH] DST_PATH [DST_SUBPATH]` prepares source/destination mount roots and runs the copier for the selected subpaths.
- `select [NODE_SELECTOR]` or `select _|NODE_SELECTOR COPY_INFO_CSV_PATH` reads copy entries from CSV, filters by node selector when provided, and runs bounded concurrent sync jobs. Use `_` only together with a custom CSV path when no selector filter is wanted.
- `start [NODE_SELECTOR|_]` or `start _|NODE_SELECTOR COPY_INFO_CSV_PATH` daemonizes selector mode using the configured pid and log files. A bare `_` keeps the default selector and default CSV path; with two arguments, `_` means no selector filter with a custom CSV path.
- `stop` reads the pid file and sends termination to the daemonized selector.

## Configuration requirements

- CLI flags must remain bound to Viper environment variables using the replacers in `main.go`.
- Default selector CSV path is `data/09_copy_entries.csv`.
- Default daemon files are `log/fast-volume-syncer.log` and `fast-volume-syncer.pid`.
- Source and destination storage mount options are configurable independently.
- Native and rsync copier paths must preserve their existing flag contracts.
- Retry settings must be controlled by `--retry-attempts`, `--retry-delay`, `--retry-max-delay`, and `--retry-max-jitter`.

## Platform requirements

- Linux supports mount and sandbox behavior.
- Unsupported-platform files must continue to compile behind build tags and return explicit unsupported behavior where applicable.
- Privileged operations such as mounting are runtime/environment requirements, not unit-test assumptions.

## Non-goals

- This repository is not a VM-management tool and does not manage unrelated guest-runtime APIs.
- Documentation must not use copied requirements from another project as current evidence.
