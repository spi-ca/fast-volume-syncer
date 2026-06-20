---
name: software-role-agents
description: Run a software-writing workflow with Pi subagents for user representative, systems engineer, designer, parallel-capable developer, implementer, QA, and reviewer roles.
---

# Software Role Agents

Use this skill to run larger fast-volume-syncer tasks through the project-local role agents.

## When to Use

- New features, bug fixes, refactors, or documentation changes need explicit requirements, implementation, QA, and review evidence.
- The task touches Go CLI code, copy, sync, or rsync orchestration, CSV/JSON CSV selector inputs, daemon pid/log handling files, or repo-local Pi resources.
- You want isolated `subagent` contexts for requirements, system constraints, design, implementation, QA, and review.

## Installed pi-subagent Frontmatter Policy

Use the current `pi-subagent` frontmatter fields only:

- `name`
- `description`
- `model`
- `thinking`
- `tools`

The project role agents use Codex model IDs and a separate `thinking` field.

## Role Responsibilities

### user-representative

- Preserve the user request, success criteria, user-facing scenarios, and blockers.
- Call out missing decisions before implementation starts.

### software-systems-engineer

- Inspect Linux runtime assumptions, file paths, sockets, permissions, external binaries, service lifecycle, and operational constraints.
- Pay extra attention when the task touches rsync execution, CSV selector inputs, storage paths, or copy/sync/select/start/stop flows.

### software-designer

- Turn the requirements and system constraints into an implementation plan.
- Separate Go code, CSV selector inputs, service files, docs, and Pi asset work into clear packages when possible.
- Define the validation strategy that matches the touched surfaces.

### software-developer

- Implement exactly one approved independent work package.
- Respect the assigned file boundaries, preserved behavior, and focused validation.
- Stop and report blockers if shared-file edits or new cross-package decisions appear.

### software-implementer

- Apply the approved plan or merge approved developer-lane results.
- Keep existing behavior outside the approved scope.
- Finish any shared-file integration and run the required focused validation.

### software-qa

- Map each acceptance criterion to concrete evidence.
- Validate happy path, edge cases, and failure handling when applicable.
- Distinguish blocking findings from non-blocking observations.

### software-reviewer

- Review correctness, maintainability, regression risk, documentation consistency, and completion evidence.
- Check that the final report matches the actual diff and command results.

## Role Input/Output Contract

| Role | Main input | Main output |
| --- | --- | --- |
| `user-representative` | User request, current docs, current behavior evidence | intent summary, acceptance criteria, user-facing scenarios, blockers |
| `software-systems-engineer` | requirements output, repo/environment evidence | system constraints, risks, feasibility, system-level validation |
| `software-designer` | requirements + system constraints + code/doc structure | change plan, work packages, risks, verification strategy |
| `software-developer` | one approved package with allowed files and acceptance criteria | package diff, focused validation, conflict report, blockers |
| `software-implementer` | approved plan, developer-lane outputs, current files | integrated changes, validation results, blockers |
| `software-qa` | acceptance criteria, diffs, validation surface | QA matrix, command/artifact evidence, findings |
| `software-reviewer` | requirements, design, diff, QA results | review verdict, blocking issues, completion audit |

## Recommended Subagent Chain

Use this chain for sufficiently large tasks. Run `parallel-development` only when the designer has separated non-overlapping packages.

```json
{
  "chain": [
    {
      "label": "requirements",
      "agent": "user-representative",
      "task": "Summarize user intent, acceptance criteria, user-facing scenarios, constraints, and blockers for: <task>"
    },
    {
      "label": "system-constraints",
      "agent": "software-systems-engineer",
      "task": "Using the requirements output, inspect repository and environment constraints and provide system-level feasibility, risks, and verification for: <task>"
    },
    {
      "label": "design",
      "agent": "software-designer",
      "task": "Using requirements and system constraints, design the implementation plan and verification strategy for: <task>"
    },
    {
      "type": "parallel",
      "label": "parallel-development",
      "tasks": [
        {
          "agent": "software-developer",
          "task": "Implement independent work package A for: <task>. Include allowed files, acceptance criteria, preserved behavior, and focused validation."
        },
        {
          "agent": "software-developer",
          "task": "Implement independent work package B for: <task>. Include allowed files, acceptance criteria, preserved behavior, and focused validation."
        }
      ]
    },
    {
      "label": "implementation-merge",
      "agent": "software-implementer",
      "task": "Integrate developer lane results, resolve approved shared-file work, and run focused validation for: <task>"
    },
    {
      "type": "parallel",
      "label": "verification-review",
      "tasks": [
        {
          "agent": "software-qa",
          "task": "Validate the implementation against acceptance criteria for: <task>"
        },
        {
          "agent": "software-reviewer",
          "task": "Review the implementation, evidence, maintainability, and completion readiness for: <task>"
        }
      ]
    }
  ],
  "mode": "spawn"
}
```

## Repo-Specific Validation Surfaces

Pick only the checks that match the changed files:

- Go source, CLI flags, config builders, or internal packages: run `gofmt -w .`, `scripts/check-go-comments.py`, `go test ./...`, tagged integration/NFS compile-only checks, and `go vet ./...`.
- CSV files: parse with Go tests, a small Python `csv` check, or another repo-approved CSV parser; JSON files: parse with `python3 -m json.tool` or `json.load`.
- daemon pid/log handling changes: run focused start/stop or pid/log-file smoke checks when safe; otherwise report the skipped runtime prerequisite clearly.
- `.pi` or docs-only changes: run `go test ./...`, parse `.pi/settings.json`, validate frontmatter for every `.pi/agents/*.md`, `.pi/skills/*/SKILL.md`, and `.pi/prompts/*.md`, run `{ printf '%s\n' README.md AGENTS.md CLAUDE.md; find docs -maxdepth 2 -type f; } | sort`, and run `git diff --check`; diagram changes also require Mermaid SVG and 2x PNG regeneration.
- Changes to copy, sync, or rsync command construction, storage paths, CSV selector inputs, or copy/sync/select/start/stop flows should include focused evidence from tests or inspected command/config outputs.

## Single-Agent Fallback

If `subagent` execution is unavailable or the task is small, the root agent should still cover the same responsibilities in order:

1. Requirements
2. System constraints
3. Design
4. Developer package work when applicable
5. Implementation/integration
6. QA
7. Review

## Quality Gates

Before finishing, confirm all of the following:

- Every required role produced output, or the root agent explicitly covered that role in a small-task fallback.
- If `software-developer` lanes were used, each lane reported allowed files, changed files, validation, and any conflict status.
- Every explicit requirement is mapped to current evidence from files, diffs, commands, tests, logs, or artifacts.
- The validation surface matches the touched files instead of using unrelated checks.
- Go changes keep related defaults and wiring aligned across CLI flags, config loading/building, and any touched docs.
- CSV/JSON or daemon edits were parsed or verified with an appropriate local command.
- Changes affecting copy, sync, or rsync copy/sync startup and daemon stop behavior include focused evidence for the resulting command/config flow.
- Existing user changes and behavior outside the approved scope were preserved.
- No unapproved shortcuts, placeholder text, dead code, duplicated logic, hidden assumptions, or undocumented behavior changes remain.
- QA and review do not leave unresolved blocking findings.

## Output Template

```markdown
## Role Results

### user-representative
- Goal:
- Acceptance criteria:
- Blockers:

### software-systems-engineer
- Constraints:
- Risks:
- System validation:

### software-designer
- Design decisions:
- Change scope:
- Verification strategy:

### software-developer
- Work packages:
- Parallel safety:
- Changed files:

### software-implementer
- Changed files:
- Key changes:
- Integration result:

### software-qa
- Commands run:
- Results:

### software-reviewer
- Audit:
- Remaining issues:
- Ready to complete:
```

## Pitfalls

- Do not skip required roles just because the task feels familiar.
- Do not parallelize packages that share files, CSV inputs, daemon files, or unresolved design decisions.
- Do not use unrelated legacy filesystem semantics as quality gates for this repo.
- Do not report unrun validations as passing.
- Do not stop at a plan if the request requires implementation and current evidence.
