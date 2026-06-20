# Pi Agents and Skills

The repository includes local Pi resources under `.pi/` for larger software-work workflows.

## Agents

- `user-representative`: preserves user intent and acceptance criteria.
- `software-systems-engineer`: checks Go runtime, filesystem, process, mount, rsync, and operational constraints.
- `software-designer`: produces implementation plans and verification strategies.
- `software-developer`: implements one isolated package of work.
- `software-implementer`: integrates approved changes and focused validation.
- `software-qa`: maps acceptance criteria to command or artifact evidence.
- `software-reviewer`: audits diffs, evidence, regressions, and completion readiness.

## Prompts and skills

- `software-role-workflow` and `software-role-agents` coordinate role-based workflows.
- `software-developer-parallel` splits non-overlapping work packages when safe.
- `iterative-findings-loop` repeats fix/QA/review cycles until blockers are cleared.
- `run-fast-volume-syncer-validation` records validation or measurement evidence for this repository.

## Maintenance rules

- Keep `.pi` instructions aligned with the actual `fast-volume-syncer` CLI and package layout.
- Do not leave copied VM-management, manifest-service, or unrelated system-service assumptions in repo-local Pi assets.
- Parse `.pi/settings.json` after edits and spot-check skill/agent frontmatter.
