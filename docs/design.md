# Design

`fast-volume-syncer` is organized around a small CLI dispatcher and four runtime modes: direct copy, sync worker, CSV selector, and daemonized selector.

## Command flow

```text
main.go
  ├─ copy   → internal/entry.Copier   → internal/copier.Runner
  ├─ sync   → internal/entry.Syncer   → internal/syncer.Runner → internal/copier.Runner
  ├─ select → internal/entry.Selector → internal/selector.Runner → sync child processes
  ├─ start  → internal/entry.DaemonStart → detached select child
  └─ stop   → internal/entry.DaemonStop
```

## Copy path

The copier scans source files, chunks work, and chooses native or rsync execution from configuration. Direct `copy` prepares the destination root with the configured private directory mode and rejects symlinked or other-user-writable path components before the backend writes beneath it. Native copying lives under `internal/copier/native`; rsync task construction and result handling live under `internal/copier/rsync`.

## Sync path

The sync runner creates a runtime area, mounts source and destination storage, opens the requested subpaths with fd-anchored `openat2` checks, bind-mounts those pinned subpaths back into the private workspace, optionally logs report data, and delegates actual data movement to the copier. Pinning the subpath roots keeps the existing native and rsync copier paths fast while preventing root-level symlink swaps after validation. Newly created destination roots use the configured private directory policy instead of inheriting broad source permissions. Namespace sandboxing is only enabled for Linux sync children launched by selector/daemon paths, where `_SYNCER_INVOKED` and `_SYNCER_SANDBOXED` are present; direct `fast-volume-syncer sync ...` runs are not sandboxed by default. `--sandbox-disabled` skips that selector-child namespace isolation only; the source and destination mount steps still require the configured runtime permissions.

## Selector path

The selector reads CSV rows into internal copy-entry values, filters entries by node selector, and invokes `sync` child processes with bounded concurrency controlled by `--worker-size`.

## Daemon path

`start` launches selector mode as a child process and records pid/log files. `stop` uses the pid file to terminate that process.

## Configuration model

`main.go` defines flags and binds them to Viper with dash/dot/underscore replacement. The `internal/entry` package reads Viper values for each command, while `internal/args` carries typed settings and assembles child-process environment variables or retry/rsync argument lists.
