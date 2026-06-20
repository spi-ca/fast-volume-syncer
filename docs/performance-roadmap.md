# Performance Roadmap

Potential performance work should remain evidence-driven and should preserve current command behavior.

## Candidate work

1. Establish representative fixtures for many-small-file and large-file copy workloads.
2. Add benchmarks for native copy, rsync chunking, file scanning, and selector fan-out.
3. Profile memory and goroutine usage for large CSV selector runs.
4. Compare `--chunk-size`, `--task-size`, and `--worker-size` settings under controlled storage conditions.
5. Improve observability around per-entry copy duration and retry counts if operational evidence shows it is needed.

## Guardrails

- Do not optimize by changing default behavior without tests and documentation.
- Do not mix local disk, NFS, and other mounted storage results in one unlabeled comparison.
- Do not introduce new dependencies or broad rewrites solely for speculative speedups.
