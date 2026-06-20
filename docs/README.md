# Documentation

This directory contains maintainer documentation for `fast-volume-syncer`.

## Project at a glance

`fast-volume-syncer` moves file trees between source and destination storage roots. It supports direct one-off copies, mounted sync jobs, CSV-driven job selection, and daemonized selector execution. The implementation is organized around a small CLI dispatcher, typed Viper-backed configuration, selector fan-out, sync mount/sandbox preparation, and native or rsync-backed copy execution.

For a visual overview, start with the project intent diagram [`diagrams/project-intent-flow.svg`](diagrams/project-intent-flow.svg), then use [`diagrams/runtime-flow.svg`](diagrams/runtime-flow.svg) and the task-specific diagrams listed in [`diagrams/README.md`](diagrams/README.md).

## Reading order

1. [`../README.md`](../README.md) - project purpose, command surface, common options.
2. [`requirements.md`](requirements.md) - supported behavior and non-goals.
3. [`design.md`](design.md) - copy/sync/select/start flow and data contracts.
4. [`architecture.md`](architecture.md) - package responsibilities.
5. [`operations.md`](operations.md) - local runbook and validation commands.
6. [`correctness-evidence.md`](correctness-evidence.md) - evidence contract for correctness-sensitive documentation claims.
7. [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md) - privileged NFS/mount sandbox evidence runbook and template.
8. [`diagrams/README.md`](diagrams/README.md) - visual navigation, Mermaid sources, and generated SVG/PNG artifacts.

## Task-specific docs

- [`commenting.md`](commenting.md) - Go comment guidance for this repository.
- [`benchmarks.md`](benchmarks.md) - benchmark and measurement rules.
- [`test-and-benchmark-gaps.md`](test-and-benchmark-gaps.md) - current test coverage and follow-up ideas.
- [`performance-roadmap.md`](performance-roadmap.md) - performance improvement backlog.
- [`pi-agents.md`](pi-agents.md) - repo-local Pi agents, prompts, and skills.
- [`guidelines/`](guidelines/) - copied agent-documentation guidance used as reference material; do not edit unless asked.

## Validation levels

1. **Baseline:** run `go test ./...` plus the document/Pi checks from [`operations.md`](operations.md).
2. **Optional Linux+bwrap integration:** on Linux hosts with usable `bwrap`, add `go test -tags=integration -run TestBwrapCopyE2E -count=1 .` and record it per [`correctness-evidence.md`](correctness-evidence.md).
3. **Privileged NFS/manual evidence:** for real NFS mount+sandbox claims, capture the target-environment runbook evidence from [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md) instead of inferring success from baseline or bwrap-only output.
