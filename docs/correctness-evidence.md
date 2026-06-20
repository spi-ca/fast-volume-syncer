# Correctness Guardrail Evidence

This file records the evidence contract for claims that are easy to overstate in `fast-volume-syncer` documentation and reviews.

## Guardrails

| Claim | Guardrail | Evidence command |
| --- | --- | --- |
| Privileged mount and network-storage behavior is environment-dependent. | Documentation must not present unit tests as proof of privileged mount or NFS behavior. | `go test ./...` runs unit coverage; privileged behavior needs the target-environment runbook in [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md) with captured output. |
| Planned/current privileged NFS integration skeletons are supplemental only. | `go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .` may be run only in a privileged environment with reachable test exports, and it does not replace captured runbook evidence unless the actual target output is included with mount/checksum/cleanup details. | `go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .` plus the filled template from [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md). |
| bwrap smoke coverage is optional and Linux-only. | `sandbox_bwrap_integration_test.go` must stay behind `//go:build integration && linux`. | `go test -tags=integration -run TestBwrapCopyE2E -count=1 .` |
| Privileged NFS skeleton coverage is opt-in and Linux-only. | `nfs_sync_sandbox_linux_integration_test.go` must stay behind `//go:build integration && nfs && linux`, require the documented `FVS_TEST_*` inputs (`FVS_TEST_SRC_NFS_HOST` through `FVS_TEST_DST_VERIFY_ROOT`), use the selector-launched sync child path with a one-row CSV, and remain supplemental to the runbook evidence. | `go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .` |
| Shared NFS evidence must not leak secrets. | The disposable selector CSV must use a dummy non-secret `source_volume_key`, and shared evidence must redact `source_volume_key`, credentials, tokens, and unrelated secrets from logs or pasted command snippets. | Filled template from [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md) or [`evidence/nfs-sync-sandbox.example.md`](evidence/nfs-sync-sandbox.example.md). |
| Benchmark claims require raw local output. | Documentation must require exact command lines and raw output before reporting benchmark results. | `go test -run '^$' -bench . -benchtime=1x ./...` or a recorded benchmark command with host notes. |
| Documentation claims must stay aligned with runnable checks. | `docs_correctness_guardrail_test.go` checks key guardrail text plus the bwrap/NFS integration build-tag contracts, and `scripts/check-go-comments.py` checks Go comment coverage. | `go test ./...` plus `scripts/check-go-comments.py` |

## Baseline evidence commands

Run these before claiming the guardrail is satisfied:

```bash
gofmt -w .
scripts/check-go-comments.py
go test ./...
go test -tags=integration -run '^$' .
go test -tags='integration,nfs' -run '^$' .
go vet ./...
git diff --check
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
```

The tagged `-run '^$'` commands are compile-only checks for opt-in integration test sources; they do not prove bwrap or privileged NFS behavior. Run optional integration evidence only on Linux hosts where `bwrap` is installed and usable:

```bash
go test -tags=integration -run TestBwrapCopyE2E -count=1 .
```

Run privileged NFS integration skeletons only on privileged target environments with reachable test exports:

```bash
go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .
```

Even when that privileged command is available, do not treat it as a substitute for the captured NFS runbook evidence unless the final report includes the actual target output alongside mount, checksum, symlink, and cleanup evidence from [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md). Keep the documented `FVS_TEST_SRC_NFS_HOST`, `FVS_TEST_SRC_NFS_EXPORT`, `FVS_TEST_DST_NFS_HOST`, `FVS_TEST_DST_NFS_EXPORT`, `FVS_TEST_SRC_SUBPATH`, `FVS_TEST_DST_SUBPATH`, `FVS_TEST_SRC_VERIFY_ROOT`, and `FVS_TEST_DST_VERIFY_ROOT` values aligned with that evidence.

## Reporting rule

A final report should name the exact commands run. If privileged mount, NFS, rsync, or benchmark behavior was not run in the target environment, report it as unverified rather than inferred from unit tests. Use [`nfs-sync-sandbox-evidence.md`](nfs-sync-sandbox-evidence.md) for manual NFS/mount sync sandbox evidence, attach actual privileged target output when citing `go test -tags='integration,nfs' -run TestNFSSyncSandboxE2E -count=1 .`, and redact `source_volume_key`, credentials, tokens, and unrelated secrets from any shared logs.
