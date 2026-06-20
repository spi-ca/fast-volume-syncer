# AGENTS.md

## Project

- Name: `fast-volume-syncer`
- Module: `amuz.es/src/spi-ca/fast-volume-syncer`
- Language: Go
- Role: copy and synchronize file trees between source and destination storage paths, with optional NFS mount orchestration, sandbox isolation, rsync mode, CSV-driven selection, and daemonized selector execution.

## Runtime model

- Binary name: `fast-volume-syncer`
- Default CSV input for `select`/`start`: `data/09_copy_entries.csv`
- Default daemon log file: `log/fast-volume-syncer.log`
- Default daemon pid file: `fast-volume-syncer.pid`
- Linux enables mount/sandbox behavior; non-Linux builds keep unsupported syscalls behind build tags.

`start` daemonizes a `select` worker process. `select` reads the CSV entries, filters by node selector when provided, and runs bounded concurrent `sync` children. `sync` mounts source and destination storage into a temporary sandbox when sandboxing is enabled, then invokes the copier. `copy` directly copies one source path to one destination path.

## CLI surface

```text
fast-volume-syncer copy SRC_PATH DST_PATH
fast-volume-syncer sync SRC_PATH [SRC_SUBPATH] DST_PATH [DST_SUBPATH]
fast-volume-syncer select [NODE_SELECTOR]
fast-volume-syncer select _|NODE_SELECTOR COPY_INFO_CSV_PATH
fast-volume-syncer start [NODE_SELECTOR|_]
fast-volume-syncer start _|NODE_SELECTOR COPY_INFO_CSV_PATH
fast-volume-syncer stop
```

For `select`, use `_` as the selector placeholder only when also passing a custom CSV path without a node selector. For `start`, a bare `_` is also accepted and means the default selector with the default CSV path. Configuration is exposed through flags and matching environment variables via Viper. Important flags include `--worker-size`, `--task-size`, `--chunk-size`, `--rsync-enabled`, `--scan-find-path`, `--sandbox-disabled`, source/destination storage mount options, and retry options.

## Editing guardrails

- Do not describe this repository with copied VM-management project identities or stacks.
- Keep docs aligned with `main.go`, `internal/entry`, `internal/selector`, `internal/syncer`, `internal/copier`, `internal/args`, and `internal/sys`.
- Keep `AGENTS.md` small per `docs/guidelines/a-complete-guide-to-agents-md.md`; put detailed runbooks in `docs/`.
- Do not modify `docs/guidelines/**` unless explicitly requested.

## Validation

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

For diagram changes also run the Mermaid SVG/2x PNG render loop from `docs/operations.md`.

For code changes also run:

```bash
gofmt -w .
scripts/check-go-comments.py
go test -tags=integration -run '^$' .
go test -tags='integration,nfs' -run '^$' .
go vet ./...
```
