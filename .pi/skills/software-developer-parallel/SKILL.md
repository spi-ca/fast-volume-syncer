---
name: software-developer-parallel
description: Split approved software work into independent packages and run parallel software-developer subagents safely.
---

# Software Developer Parallel

Use this skill to split approved fast-volume-syncer work into independent packages and run `software-developer` lanes in parallel only when the file scopes and decisions do not overlap.

## When to Use

- The task can be separated into independent Go packages, docs/Pi updates, CSV fixtures, or runtime configuration changes.
- Each lane can receive explicit allowed files, acceptance criteria, preserved behavior, and focused validation.
- Parallel work will reduce elapsed time without creating merge-risk on shared files.

Do not use this skill when:

- The same files, structs, command builders, or docs must be edited by multiple lanes.
- One package depends on another package's unfinished result.
- The design is still unresolved and the work needs sequential decision-making first.

## Agent Configuration

`software-developer` lanes use:

- Model: `openai-codex/gpt-5.4`
- Thinking: `high`
- Tools: `read`, `grep`, `find`, `ls`, `bash`, `edit`, `write`

## Procedure

1. Read the task, repository instructions, approved design, and current user changes.
2. Propose package boundaries by responsibility and file ownership.
3. For each package, define:
   - goal and acceptance criteria
   - allowed files or directories
   - behavior that must be preserved
   - focused validation commands
   - files that must stay out of scope
4. Keep shared-file edits in a sequential merge step instead of parallel lanes.
5. Run only the truly independent packages with `subagent` parallel mode.
6. Collect each lane's changed files, validation results, and blocker/conflict report.
7. Merge shared-file work in a serial step with the root agent or `software-implementer`.
8. Run QA and review after integration.
9. Repeat only the affected lanes if findings require rework.
10. Finish only when all explicit requirements map to current evidence.

## Repo-Specific Package Examples

Examples that are often safe to split:

- one lane for Go code under `internal/**` or `main.go`
- one lane for repo-local Pi docs/resources under `.pi/**` and `docs/pi-agents.md`
- one lane for selector CSV fixtures, docs, or daemon/runtime configuration examples

Examples that are usually not safe to split without a merge step:

- multiple lanes editing the same Go package, shared config structs, or the same `.pi` skill file
- one lane changing command construction while another lane edits the docs that explain the same command surface

## Parallel Subagent Template

```json
{
  "tasks": [
    {
      "agent": "software-developer",
      "task": "Implement work package A for <task>. Allowed files: <paths>. Acceptance criteria: <criteria>. Preserve existing behavior outside this package. Run focused validation: <commands>. Report blockers and parallel-safety conflicts."
    },
    {
      "agent": "software-developer",
      "task": "Implement work package B for <task>. Allowed files: <paths>. Acceptance criteria: <criteria>. Preserve existing behavior outside this package. Run focused validation: <commands>. Report blockers and parallel-safety conflicts."
    }
  ],
  "mode": "spawn"
}
```

## Validation Guidance

Match validation to the touched files:

- Go code: `gofmt -w .`, `scripts/check-go-comments.py`, `go test ./...`, tagged integration/NFS compile-only checks, and `go vet ./...`
- CSV/JSON: parse the touched CSV or JSON files with a local parser
- daemon pid/log handling: focused start/stop or pid/log-file smoke checks when safe
- `.pi`/docs-only work: `go test ./...` (including guardrails), parse `.pi/settings.json`, validate frontmatter for every `.pi/agents/*.md`, `.pi/skills/*/SKILL.md`, and `.pi/prompts/*.md`, run `{ printf '%s\n' README.md AGENTS.md CLAUDE.md; find docs -maxdepth 2 -type f; } | sort`, and run `git diff --check`; regenerate Mermaid SVG/2x PNG when diagrams change

## Quality Gates

- Each lane's changed files stay within its allowed files.
- No two lanes edit the same file at the same time.
- Each package has focused validation that matches its surface.
- Go/config/service changes receive the appropriate repo-native checks.
- QA and review do not leave unresolved blocking findings.
- No unapproved shortcuts, placeholder text, dead code, duplicated logic, or undocumented behavior changes remain.

## Output Template

```markdown
## Parallel Developer Plan

| Package | Agent | Allowed files | Acceptance criteria | Validation |
| --- | --- | --- | --- | --- |
| ... | `software-developer` | ... | ... | ... |

## Lane Results
- Package:
  - Files changed:
  - Validation:
  - Blockers/conflicts:

## Merge and Review
- Integrated files:
- QA evidence:
- Review verdict:
- Remaining findings/blockers:
```

## Pitfalls

- Do not force artificial package boundaries just to use parallelism.
- Do not assign the same file to multiple lanes.
- Do not treat one lane's validation as proof for another lane.
- Do not bypass project-local subagent confirmation.
- Do not use unrelated legacy benchmark expectations for fast-volume-syncer work.
