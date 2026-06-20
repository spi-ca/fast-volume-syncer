# Diagrams

Mermaid diagrams in this directory describe the current `fast-volume-syncer` project intent, copy/sync/select/start architecture, NFS evidence flow, validation surfaces, and generated visual artifacts.

## Visual navigation

| Question | Diagram | Rendered artifacts |
| --- | --- | --- |
| Why does this project exist and what operating problem does it solve? | [`project-intent-flow.mmd`](project-intent-flow.mmd) | [`svg`](project-intent-flow.svg), [`png`](project-intent-flow.png) |
| How is the repository organized across CLI, entry adapters, runtime packages, support packages, and platform boundaries? | [`project-architecture-overview.mmd`](project-architecture-overview.mmd) | [`svg`](project-architecture-overview.svg), [`png`](project-architecture-overview.png) |
| How do CLI commands map to entry adapters and core runners? | [`runtime-flow.mmd`](runtime-flow.mmd) | [`svg`](runtime-flow.svg), [`png`](runtime-flow.png) |
| How do flags and environment values reach child processes? | [`configuration-flow.mmd`](configuration-flow.mmd) | [`svg`](configuration-flow.svg), [`png`](configuration-flow.png) |
| How does the copier scan, chunk, retry, and choose native vs rsync? | [`copier-execution-flow.mmd`](copier-execution-flow.mmd) | [`svg`](copier-execution-flow.svg), [`png`](copier-execution-flow.png) |
| How do `start` and `stop` manage daemonized selector execution? | [`daemon-start-stop-flow.mmd`](daemon-start-stop-flow.mmd) | [`svg`](daemon-start-stop-flow.svg), [`png`](daemon-start-stop-flow.png) |
| How does selector CSV input become sync child work? | [`selector-csv-flow.mmd`](selector-csv-flow.mmd) | [`svg`](selector-csv-flow.svg), [`png`](selector-csv-flow.png) |
| What does `sync` do with sandboxing, mounts, reports, and copier delegation? | [`syncer-sandbox-flow.mmd`](syncer-sandbox-flow.mmd) | [`svg`](syncer-sandbox-flow.svg), [`png`](syncer-sandbox-flow.png) |
| Which validation checks apply to docs, `.pi`, diagrams, and Go code? | [`validation-checks.mmd`](validation-checks.mmd) | [`svg`](validation-checks.svg), [`png`](validation-checks.png) |
| When can NFS/mount sandbox behavior be claimed as verified with the [runbook](../nfs-sync-sandbox-evidence.md)? | [`nfs-sync-sandbox-evidence-flow.mmd`](nfs-sync-sandbox-evidence-flow.mmd) | [`svg`](nfs-sync-sandbox-evidence-flow.svg), [`png`](nfs-sync-sandbox-evidence-flow.png) |

## Sources

The `.mmd` files are the source of truth. Update the matching SVG and 2x PNG artifacts whenever a source changes.

## Export command

Use Mermaid CLI with the local Puppeteer sandbox config. PNG artifacts are exported at 2x scale.

```bash
for src in docs/diagrams/*.mmd; do
  base="${src%.mmd}"
  npx --yes @mermaid-js/mermaid-cli -i "$src" -o "$base.svg" -p docs/diagrams/puppeteer-config.json -b white
  npx --yes @mermaid-js/mermaid-cli -i "$src" -o "$base.png" -p docs/diagrams/puppeteer-config.json -b white -s 2
done
```

## Artifact policy

Generated SVG/PNG files are required for committed diagram sources. Regenerate them from the matching `.mmd` source in the same change and do not leave stale diagrams from other projects.
