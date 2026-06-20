# Operations and Validation

## Prerequisites

- Go toolchain compatible with `go.mod`.
- `rsync` when `--rsync-enabled` is used.
- Linux privileges and mount helpers for `sync` runs that mount source and destination storage. Direct `sync` runs are not namespace-sandboxed by default; selector/daemon-launched Linux sync children may enter the sandbox path. `--sandbox-disabled` disables that selector-child namespace isolation, not the mount steps.
- A CSV file matching the selector entry format when using `select` or `start`.

## Standard validation

Docs or Pi-resource changes. The `go test ./...` baseline includes repository guardrail tests that transitively run the comment checker and tagged integration/NFS compile-only checks, so those tools and build-tagged test sources must be valid even for documentation-only edits.

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

Diagram source changes also require regenerating matching SVG and 2x PNG artifacts:

```bash
for src in docs/diagrams/*.mmd; do
  base="${src%.mmd}"
  npx --yes @mermaid-js/mermaid-cli -i "$src" -o "$base.svg" -p docs/diagrams/puppeteer-config.json -b white
  npx --yes @mermaid-js/mermaid-cli -i "$src" -o "$base.png" -p docs/diagrams/puppeteer-config.json -b white -s 2
done
```

Code changes should also run these commands explicitly so failures are easy to triage:

```bash
gofmt -w .
scripts/check-go-comments.py
go test ./...
go test -tags=integration -run '^$' .
go test -tags='integration,nfs' -run '^$' .
go vet ./...
```

Optional bwrap integration smoke (Linux hosts with usable `bwrap` only; excluded from normal builds/tests by build tags; see [`correctness-evidence.md`](correctness-evidence.md)):

```bash
go test -tags=integration -run TestBwrapCopyE2E -count=1 .
```

Privileged NFS integration is a separate target-environment check. If the committed NFS skeleton is available, run `go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .` only in a privileged environment with reachable test exports. That command does not replace the captured runbook evidence unless the actual target output, mount details, and cleanup results are included in the report. The committed skeleton uses the selector-launched `select 0 <csv>` path, the documented `FVS_TEST_SRC_NFS_HOST` through `FVS_TEST_DST_VERIFY_ROOT` inputs, and a disposable one-row CSV.

Manual NFS/mount sync sandbox evidence requires a privileged target environment and the real selector-launched child path documented in [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md). Use a dummy non-secret `source_volume_key` in disposable selector CSVs and redact `source_volume_key`, credentials, tokens, and unrelated secrets from any shared logs.

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
- Expect direct `copy` destination roots to use the configured private directory policy and reject symlinked or other-user-writable path components before copying.
- Expect newly created destination roots to use the configured private directory policy rather than inheriting broad source permissions.
- Confirm `--worker-size`, `--task-size`, and `--chunk-size` are appropriate for host IO capacity.
- Capture exact command lines, allowlisted relevant environment variables, stdout/stderr, and timing when collecting performance evidence; do not record unrelated secrets or inherited credentials.
- Use only disposable NFS selector CSVs for privileged evidence. If the CSV includes `source_volume_key`, it must be a dummy non-secret placeholder.
- Redact `source_volume_key`, credentials, tokens, and unrelated secrets before sharing logs or evidence captured from privileged runs.
- Do not claim privileged mount, selector-child sandbox, pinned subpath bind-mount, or rsync behavior was verified unless that command was actually run in the target environment.
- For privileged daemon or report-enabled runs on shared hosts, set `--log-file` and `--pid-file` under a dedicated root-owned `0700` directory; report logs can include paths, mount metadata, and capacity information.
