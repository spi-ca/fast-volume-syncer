# Benchmarks and Measurement

This repository does not currently maintain a formal benchmark evidence bundle. Treat measurements as local evidence until they include reproducible commands, environment details, and raw output.

## Candidate local benchmarks

- File scanning throughput under `internal/copier/find`.
- Native copier throughput for representative file sizes and counts.
- Rsync task overhead with `--rsync-enabled` and varied chunk sizes.
- Selector fan-out behavior with different `--worker-size` values.
- Retry overhead for transient copy failures.

## Measurement rules

- Record the exact command line and relevant flags/environment variables.
- Separate local filesystem tests from mounted storage or network storage tests.
- Include host details that affect IO results.
- Re-run correctness checks after changing implementation code.
- Do not reuse copied benchmark claims from another project as evidence for this repository.

## Useful commands

```bash
go test -bench=. -benchmem ./...
/usr/bin/time -v go test ./...
```
