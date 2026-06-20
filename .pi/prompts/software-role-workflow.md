---
description: Run software work through user representative, systems engineer, designer, parallel-capable developer, implementer, QA, and reviewer subagents
argument-hint: "<task>"
---
Reminder: This file is a `.pi/prompts/` prompt template. `$ARGUMENTS` is the remaining user input after the prompt command or skill invocation.

Run this software-writing task through the project-local role agents:

Task:
$ARGUMENTS

Use the `subagent` tool with a chain when available. Use `mode: "spawn"` unless the current conversation context is required.

Role set:
1. `user-representative` — preserve user intent, acceptance criteria, user-facing scenarios, and blockers.
2. `software-systems-engineer` — inspect OS/runtime/dependency/permission/deployment/operational constraints and system-level verification.
3. `software-designer` — produce implementation design, change scope, risks, work packages, and verification strategy.
4. `software-developer` — conditionally implement one independent work package with focused validation; multiple lanes may run in parallel when file scopes do not overlap.
5. `software-implementer` — apply or integrate approved changes and run focused validation.
6. `software-qa` — validate acceptance criteria with tests/checks/smoke evidence.
7. `software-reviewer` — review diff, quality, regressions, and completion evidence.

Required for every task: user representative, systems engineer when system constraints matter, designer, implementer, QA, and reviewer. Use `software-developer` only when the design identifies at least one independent work package; otherwise document why it was skipped and implement sequentially.

When defining validation commands for this repo, use `go test ./...` as the baseline check. Add touched-surface checks that apply: CSV fixture inspection for selector changes, JSON/frontmatter/inventory checks and `git diff --check` for `.pi` or docs resources, and daemon pid/log smoke checks when start/stop behavior changes.

Recommended chain shape:

```json
{
  "chain": [
    {
      "label": "requirements",
      "agent": "user-representative",
      "task": "Summarize user intent, acceptance criteria, user-facing scenarios, constraints, and blockers for: $ARGUMENTS"
    },
    {
      "label": "system-constraints",
      "agent": "software-systems-engineer",
      "task": "Using prior outputs, inspect repository/environment constraints and provide system-level feasibility, risks, and verification for: $ARGUMENTS"
    },
    {
      "label": "design",
      "agent": "software-designer",
      "task": "Using prior outputs, design the implementation plan and verification strategy for: $ARGUMENTS"
    },
    {
      "type": "parallel",
      "label": "parallel-development",
      "tasks": [
        {
          "agent": "software-developer",
          "task": "Implement independent work package A for: $ARGUMENTS. Allowed files: <paths>. Acceptance criteria: <criteria>. Preserve existing behavior outside this package. Run focused validation: <commands>."
        },
        {
          "agent": "software-developer",
          "task": "Implement independent work package B for: $ARGUMENTS. Allowed files: <paths>. Acceptance criteria: <criteria>. Preserve existing behavior outside this package. Run focused validation: <commands>."
        }
      ]
    },
    {
      "label": "implementation-merge",
      "agent": "software-implementer",
      "task": "Integrate developer lane results or implement remaining approved changes for: $ARGUMENTS"
    },
    {
      "type": "parallel",
      "label": "verification-review",
      "tasks": [
        {
          "agent": "software-qa",
          "task": "Validate the implementation against acceptance criteria for: $ARGUMENTS"
        },
        {
          "agent": "software-reviewer",
          "task": "Review the implementation, evidence, maintainability, and completion readiness for: $ARGUMENTS"
        }
      ]
    }
  ],
  "mode": "spawn"
}
```

Use the `parallel-development` stage only when the designer has separated non-overlapping work packages. If there is exactly one independent package, run one `software-developer` lane before `software-implementer` integration. If the work cannot be safely split into a developer package, skip `software-developer` and use `software-implementer` sequentially.

If project-local subagent confirmation is required, do not try to bypass it. Ask for confirmation or fall back to role-by-role execution in the root agent.

Do not finish until every explicit requirement is mapped to current evidence from files, commands, diffs, tests, logs, or artifacts. If validation fails, triage and fix the cause before final reporting.
