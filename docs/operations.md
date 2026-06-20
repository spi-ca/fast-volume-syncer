# Operations and Validation

## Prerequisites

- Go toolchain compatible with `go.mod`.
- `rsync` when `--rsync-enabled` is used.
- Linux privileges and mount helpers for `sync` runs that mount source and destination storage. `--sandbox-disabled` disables namespace isolation, not the mount steps.
- A CSV file matching the selector entry format when using `select` or `start`.

## Standard validation

Docs or Pi-resource changes:

```bash
go test ./...
{ printf '%s\n' README.md AGENTS.md CLAUDE.md; find docs -maxdepth 2 -type f; } | sort
python3 - <<'PY'
import json, pathlib
json.load(open('.pi/settings.json'))
for p in list(pathlib.Path('.pi/agents').glob('*.md')) + list(pathlib.Path('.pi/skills').glob('*/SKILL.md')) + list(pathlib.Path('.pi/prompts').glob('*.md')):
    s = p.read_text()
    if not s.startswith('---\n') or '\n---\n' not in s[4:]:
        raise SystemExit(f'bad frontmatter: {p}')
print('ok')
PY
git diff --check
```

Code changes:

```bash
gofmt -w .
go vet ./...
go test ./...
```

## Local command examples

Direct copy:

```bash
go run . copy /path/source /path/destination
```

Sync with explicit storage roots:

```bash
go run . sync /mnt/source /mnt/destination
```

Sync selected subpaths:

```bash
go run . sync /mnt/source project-a /mnt/destination project-a
```

Run selector with defaults:

```bash
go run . select
```

Run selector for one node and CSV path:

```bash
go run . select 3 data/09_copy_entries.csv
# use _ to provide a custom CSV path without filtering by node selector
go run . select _ data/09_copy_entries.csv
```

Daemonize selector and stop it later:

```bash
go run . start 3 data/09_copy_entries.csv
go run . stop
```

## Operational checks

- Confirm source and destination mount hosts/options before privileged sync runs.
- Confirm `--worker-size`, `--task-size`, and `--chunk-size` are appropriate for host IO capacity.
- Capture exact command lines, relevant environment variables, stdout/stderr, and timing when collecting performance evidence.
- Do not claim privileged mount or rsync behavior was verified unless that command was actually run in the target environment.
