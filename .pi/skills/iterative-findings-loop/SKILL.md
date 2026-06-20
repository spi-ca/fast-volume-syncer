---
name: iterative-findings-loop
description: Repeat implementation or documentation updates through subagent QA and review until findings and blockers are cleared with current evidence.
---

# Iterative Findings Loop

Use this skill when QA or review findings require one or more fix-and-revalidate loops before a fast-volume-syncer task can be completed.

## When to Use

- QA, review, or focused validation found blocking issues.
- The task spans Go code, YAML/service files, docs, or Pi assets and will likely need more than one correction pass.
- You need to preserve a clear trail of findings, fixes, and fresh evidence.

## Procedure

1. Restate the current completion criteria from the latest user request.
2. Classify findings as blocking, non-blocking, or unclear.
3. Keep implementation, QA, and review outputs separate.
4. Re-scope the fix work:
   - use parallel `software-developer` lanes only for non-overlapping file scopes
   - use `software-implementer` or the root agent for shared-file integration
5. Apply the smallest coherent fix for the current findings.
6. Re-run the focused validation immediately after the fix.
7. Re-run `software-qa` and `software-reviewer` when the fix changes behavior or evidence.
8. Update the finding list with fresh status and remaining risk.
9. Repeat until no blocking findings remain and every explicit requirement maps to current evidence.

## Repo-Specific Revalidation Guide

Choose the checks that match the touched files:

- Go code: `go test ./...`
- CSV files: parse with Go tests, Python `csv`, or another repo-approved CSV parser; JSON files: parse with `python3 -m json.tool` or `json.load`
- daemon pid/log handling changes: run focused start/stop or pid/log-file smoke checks when safe
- `.pi`/docs-only changes: `go test ./...`, JSON parse, frontmatter spot check, inventory listings, `git diff --check`

If a required local tool is unavailable, report that limitation explicitly instead of marking the check as passed.

## Parallel Guidance

- Parallelize only when file ownership and decisions do not overlap.
- QA and review are good parallel candidates after implementation stabilizes.
- If multiple findings point to the same file or config surface, fix them in one sequential step.

## Pitfalls

- Do not stop after listing findings; connect them to concrete fixes and fresh validation.
- Do not report stale QA or review results as current evidence.
- Do not leave blocking findings unresolved just because a non-blocking subset passed.
- Do not reintroduce unrelated legacy validation language for this repo.

## Verification

- The latest QA pass has no blocking failures.
- The latest review pass does not request further changes.
- Every explicit requirement maps to current evidence.
- The final diff reflects the claimed fixes.
