# fast-volume-syncer

`fast-volume-syncer` is a Go command-line tool for copying and synchronizing file trees between source and destination storage paths. It can run a single copy, mount source/destination storage and run a sync worker, fan out sync jobs from a CSV selector, or daemonize that selector.

## Commands

```text
fast-volume-syncer copy SRC_PATH DST_PATH
fast-volume-syncer sync SRC_PATH [SRC_SUBPATH] DST_PATH [DST_SUBPATH]
fast-volume-syncer select [NODE_SELECTOR]
fast-volume-syncer select _|NODE_SELECTOR COPY_INFO_CSV_PATH
fast-volume-syncer start [NODE_SELECTOR|_]
fast-volume-syncer start _|NODE_SELECTOR COPY_INFO_CSV_PATH
fast-volume-syncer stop
```

Default selector CSV path: `data/09_copy_entries.csv`. To pass a custom CSV path without filtering by node selector, use `_` as the first argument together with the CSV path, for example `fast-volume-syncer select _ custom.csv`. In `start`, a bare `_` is also accepted and means the default selector with the default CSV path.

## Common options

- `--worker-size`: maximum concurrent `sync` child processes for selector mode.
- `--task-size`, `--chunk-size`: copier batching and rsync concurrency controls.
- `--rsync-enabled` plus `--rsync-*`: use rsync-backed copying instead of the native copier path.
- `--scan-find-path`: `find` binary path, or Go scanning implementation when configured by the copier package.
- `--sandbox-disabled`: skip Linux namespace/mount sandbox isolation.
- `--src-storage-*`, `--dst-storage-*`: source and destination storage mount host, option, and mount-name values.
- `--retry-*`: retry attempts, delay, max delay, and jitter.

Flags are also bound to environment variables through Viper using uppercase names with separators converted to underscores.

## How it works

The usual high-level flow is:

1. `copy` runs the copier directly for one source/destination pair.
2. `sync` prepares the configured storage mounts, optionally enters sandbox isolation, and delegates file movement to the copier.
3. `select` reads CSV entries, filters by node selector when requested, and fans out bounded concurrent `sync` child processes.
4. `start` daemonizes selector mode with configured pid/log files; `stop` reads the pid file and sends `SIGTERM`.

See [`docs/diagrams/README.md`](docs/diagrams/README.md) for SVG/PNG diagrams of the runtime, configuration, copier, selector, syncer, daemon, and validation flows.

## Documentation

Start with [`docs/README.md`](docs/README.md) for maintainer documentation, architecture notes, operations, diagrams, Pi resources, and guideline references.

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

For code changes:

```bash
gofmt -w .
go vet ./...
```
