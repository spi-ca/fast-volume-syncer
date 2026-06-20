---
name: run-fast-volume-syncer-validation
description: Run and record fast-volume-syncer validation or measurement evidence for copy, sync, selector, daemon, docs, or Pi-resource changes.
---

# Repo Validation and Measurement Helper

Use this skill to choose and record validation evidence for the current `fast-volume-syncer` repository.

## When to Use

- A change touches Go CLI dispatch, copy/sync/select/start/stop behavior, copier implementations, selector CSV handling, sandbox or mount behavior, retry logic, docs, or `.pi` resources.
- Review needs reproducible validation or timing evidence beyond a plain statement that the code looks correct.
- You need a compact evidence bundle for current repository behavior.

Do not use this skill for evidence from unrelated projects or for validation surfaces that are not present in this repository.

## Procedure

1. Pick validation surfaces that match changed files.
   - Go code, CLI flags, copier/syncer/selector logic, or args structs: run `go test ./...`.
   - Code changes: run `gofmt -w .` before tests and `go vet ./...` after formatting.
   - Docs or `.pi` changes: run `go test ./...`, parse JSON files, spot-check frontmatter, run the docs inventory command, and run `git diff --check`.
2. If timing evidence is requested, wrap the exact repo-native command with a non-root timing tool such as `/usr/bin/time` and record the full command line.
3. Prefer focused measurements around file scanning, native copy, rsync task execution, selector fan-out, or daemon pid/log behavior.
4. Keep raw command output as the source of truth and write summaries separately.
5. Record environment assumptions that affect the result: OS, privilege level, rsync availability, storage type, current worktree state, and skipped checks.
6. Re-run correctness checks after performance or timing experiments if implementation changed.

## Pitfalls

- Do not require root-only mount workflows for ordinary unit validation.
- Do not assume `rsync`, mount privileges, or network storage are available; report missing tools or privileges explicitly.
- Do not reuse copied sync job-management validation language or benchmark claims.
- Do not claim timing evidence without exact command line and environment notes.

## Verification

- Touched Go code passes `go test ./...` and, when code changed, `go vet ./...` after `gofmt -w .`.
- Docs/`.pi` changes pass `go test ./...`, JSON/frontmatter spot checks, the docs inventory command, and `git diff --check`.
- Any recorded measurement includes command lines, environment notes, and corresponding correctness checks.
