---
description: Split software work into independent packages and run parallel software-developer subagents
argument-hint: "<task>"
---
Reminder: This file is a `.pi/prompts/` prompt template. `$ARGUMENTS` is the remaining user input after the prompt command or skill invocation.

Run this software development task with parallel-capable developer lanes when safe:

Task:
$ARGUMENTS

Use the `software-developer-parallel` skill as the workflow guide.

Required behavior:
1. Inspect the request, repository instructions, existing docs, and relevant implementation.
2. Decide whether the task can be split into independent work packages.
3. For every package, define allowed files, acceptance criteria, preserved behavior, and focused validation.
4. If packages overlap on files or decisions, do not parallelize that overlap; use a sequential merge step.
5. Run independent packages with `subagent` parallel mode using `software-developer` agents when available.
6. After implementation, inspect diffs and validation evidence.
7. Run QA and review, preferably with `software-qa` and `software-reviewer` in a parallel verification stage.
8. Repeat fixes until findings and blockers are cleared.

When composing validation commands for this repo, follow `docs/operations.md`: use `go test ./...` as the baseline (it transitively runs comment and tagged compile guardrails), run `gofmt -w .`, `scripts/check-go-comments.py`, tagged integration/NFS compile-only checks, and `go vet ./...` for code changes, regenerate Mermaid SVG/2x PNG for diagram changes, and add JSON/frontmatter/inventory checks plus `git diff --check` for `.pi` or docs resources.

Recommended parallel implementation shape:

```json
{
  "tasks": [
    {
      "agent": "software-developer",
      "task": "Implement package 1 for: $ARGUMENTS. Allowed files: <paths>. Acceptance criteria: <criteria>. Preserve existing behavior outside this package. Run focused validation: <commands>. Report parallel-safety conflicts."
    },
    {
      "agent": "software-developer",
      "task": "Implement package 2 for: $ARGUMENTS. Allowed files: <paths>. Acceptance criteria: <criteria>. Preserve existing behavior outside this package. Run focused validation: <commands>. Report parallel-safety conflicts."
    }
  ],
  "mode": "spawn"
}
```

Recommended verification shape:

```json
{
  "type": "parallel",
  "label": "qa-review",
  "tasks": [
    {
      "agent": "software-qa",
      "task": "Validate implemented changes against acceptance criteria for: $ARGUMENTS"
    },
    {
      "agent": "software-reviewer",
      "task": "Review diffs, evidence, maintainability, and completion readiness for: $ARGUMENTS"
    }
  ]
}
```

If project-local subagent confirmation is required, do not try to bypass it. Ask for confirmation or perform the same role-by-role workflow in the root agent.

Do not finish until every explicit requirement is mapped to current evidence from files, commands, diffs, tests, logs, or artifacts. If validation or review finds blockers, update the plan, fix the cause, and re-run the relevant checks.
