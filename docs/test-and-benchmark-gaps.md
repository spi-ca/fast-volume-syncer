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
- optional `integration && linux` bwrap copy E2E smoke coverage in `sandbox_bwrap_integration_test.go`.
- optional `integration && nfs && linux` privileged NFS sync sandbox skeleton coverage in `nfs_sync_sandbox_linux_integration_test.go`.

Use `go test ./...` as the baseline regression command, and add compile-only tagged checks `go test -tags=integration -run '^$' .` plus `go test -tags='integration,nfs' -run '^$' .` when root-level integration test sources change. Use `go test -tags=integration -run TestBwrapCopyE2E -count=1 .` for the optional bwrap integration smoke when the host supports bwrap. Run the privileged NFS sync sandbox skeleton as `go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .` only on privileged hosts with reachable test exports, and treat any resulting claim as supplemental until the captured evidence template from [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md) or [`evidence/nfs-sync-sandbox.example.md`](evidence/nfs-sync-sandbox.example.md) is filled with target output.

## Useful follow-up tests

- Broader CLI argument validation for `copy`, `sync`, `start`, and `stop`.
- Viper flag/environment binding precedence for important options.
- Sync runner behavior for direct unsandboxed runs, selector-launched sandbox runs, sandbox-disabled selector children, and mount failures stubbed or isolated.
- Daemon start/stop pid-file edge cases.
- Copier retry behavior across native and rsync paths.
- CSV parsing edge cases for quoted commas, CRLF input, cancellation, and scanner/log behavior.

## Diagram documentation

Current Mermaid sources live in [`diagrams/`](diagrams/):

- `project-intent-flow.mmd` captures the operator problem and high-level project intent.
- `project-architecture-overview.mmd` captures CLI, entry adapters, runtime packages, system helpers, and validation boundaries.
- `runtime-flow.mmd` captures command dispatch and package-level runtime flow.
- `configuration-flow.mmd` captures flags/env/Viper binding through entrypoints, argument structs, and child-process environments.
- `copier-execution-flow.mmd` captures directory preparation, scanning, chunking, native/rsync selection, retry, and result aggregation.
- `daemon-start-stop-flow.mmd` captures daemonized selector startup, child environment, pid/log handling, and stop signal flow.
- `selector-csv-flow.mmd` captures CSV parsing, selector filtering, and sync child fan-out.
- `syncer-sandbox-flow.mmd` captures direct sync and selector-launched child flow through conditional sandboxing, mount setup, optional reports, copier selection, and cleanup.
- `nfs-sync-sandbox-evidence-flow.mmd` captures the privileged evidence collection loop.
- `validation-checks.mmd` captures validation surfaces for docs, `.pi`, diagrams, and Go code.

Generated SVG/PNG artifacts are required for every committed `.mmd` source and must be regenerated from matching sources whenever a diagram changes.

## Benchmark gaps

- Native copier throughput by file count and file size.
- Rsync mode throughput and process overhead by `--chunk-size` and `--task-size`.
- Selector fan-out scaling by `--worker-size`.
- Storage-backed sync timing with source/destination mount details recorded.
- Routine privileged target-environment execution evidence for the committed `integration,nfs` sync sandbox skeleton.

## Correctness guardrail

`docs_correctness_guardrail_test.go` keeps the documentation claims above tied to executable safeguards: privileged mount behavior must remain described as environment-dependent, bwrap E2E must stay behind `integration && linux`, the NFS sync sandbox skeleton must stay behind `integration && nfs && linux` with its documented `FVS_TEST_*` inputs and selector-launched one-row CSV flow, and local benchmark claims must require raw command output. Shared NFS evidence must require a dummy non-secret `source_volume_key` plus redaction of `source_volume_key` and other secrets. The evidence contract is recorded in [`correctness-evidence.md`](correctness-evidence.md).

## Reporting rules

- Do not claim benchmark results without raw command output.
- Do not treat privileged mount or network storage behavior as verified by unit tests alone.
- Do not treat `go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .` as sufficient evidence unless the actual privileged target output is captured with the example/template evidence.
- Use a dummy non-secret `source_volume_key` in disposable NFS selector CSVs, and redact `source_volume_key`, credentials, tokens, and unrelated secrets from shared logs.
- Keep this file aligned with the actual tests present under `internal/**` and root-level CLI tests.
