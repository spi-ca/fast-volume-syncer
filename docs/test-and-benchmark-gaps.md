# Test and Benchmark Gaps

## Current automated coverage

Existing tests cover selected behavior in these areas:

- CLI usage output for command forms and selector CSV placeholder guidance.
- CLI rejection of a bare custom CSV path for `select`, matching the documented `_` placeholder requirement.
- Argument/environment assembly under `internal/args`, including rsync flags, copier retry/scan settings, and syncer mount environment propagation.
- Execution result stderr ring-buffer/error formatting under `internal/returns`.
- CSV selector parsing under `internal/selector`, including header skipping, malformed-row skipping, field trimming, numeric parsing, and node-selector filtering.
- file discovery under `internal/copier/find`.
- native copier behavior.
- rsync task construction.
- selector runner behavior.
- file mode conversion in `internal/sys`.
- string/unit conversion helpers in `internal/util`.

Use `go test ./...` as the baseline regression command.

## Useful follow-up tests

- Broader CLI argument validation for `copy`, `sync`, `start`, and `stop`.
- Viper flag/environment binding precedence for important options.
- Sync runner behavior with sandbox disabled and with mount failures stubbed or isolated.
- Daemon start/stop pid-file edge cases.
- Copier retry behavior across native and rsync paths.
- CSV parsing edge cases for quoted commas, CRLF input, cancellation, and scanner/log behavior.

## Diagram documentation

Current Mermaid sources live in [`diagrams/`](diagrams/):

- `runtime-flow.mmd` captures command dispatch and package-level runtime flow.
- `configuration-flow.mmd` captures flags/env/Viper binding through entrypoints, argument structs, and child-process environments.
- `copier-execution-flow.mmd` captures directory preparation, scanning, chunking, native/rsync selection, retry, and result aggregation.
- `daemon-start-stop-flow.mmd` captures daemonized selector startup, child environment, pid/log handling, and stop signal flow.
- `selector-csv-flow.mmd` captures CSV parsing, selector filtering, and sync child fan-out.
- `syncer-sandbox-flow.mmd` captures sync command flow through sandbox/mount setup, optional reports, copier selection, and cleanup.
- `validation-checks.mmd` captures validation surfaces for docs, `.pi`, diagrams, and Go code.

Generated SVG/PNG artifacts are optional and should only be committed when regenerated from matching `.mmd` sources.

## Benchmark gaps

- Native copier throughput by file count and file size.
- Rsync mode throughput and process overhead by `--chunk-size` and `--task-size`.
- Selector fan-out scaling by `--worker-size`.
- Storage-backed sync timing with source/destination mount details recorded.

## Reporting rules

- Do not claim benchmark results without raw command output.
- Do not treat privileged mount or network storage behavior as verified by unit tests alone.
- Keep this file aligned with the actual tests present under `internal/**` and root-level CLI tests.
