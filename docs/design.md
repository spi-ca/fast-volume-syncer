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

The copier scans source files, chunks work, and chooses native or rsync execution from configuration. Native copying lives under `internal/copier/native`; rsync task construction and result handling live under `internal/copier/rsync`.

## Sync path

The sync runner optionally enters sandbox isolation, then creates a runtime area, mounts source and destination storage, optionally logs report data, and delegates actual data movement to the copier. `--sandbox-disabled` skips namespace isolation only; the source and destination mount steps still require the configured runtime permissions.

## Selector path

The selector reads CSV rows into internal copy-entry values, filters entries by node selector, and invokes `sync` child processes with bounded concurrency controlled by `--worker-size`.

## Daemon path

`start` launches selector mode as a child process and records pid/log files. `stop` uses the pid file to terminate that process.

## Configuration model

`main.go` defines flags and binds them to Viper with dash/dot/underscore replacement. Internal argument structs under `internal/args` read those values and pass typed settings to copier, syncer, selector, and retry logic.
