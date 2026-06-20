# Commenting Guidance

Use comments to explain non-obvious storage, process, platform, or retry behavior. Avoid comments that merely restate function names or straightforward assignments.

## Expectations

- Exported Go identifiers should have useful comments when they are part of a package contract.
- Platform-specific files should explain build-tag behavior only when it affects callers.
- Risky operational assumptions, such as mount privileges or child-process cleanup, deserve comments near the code that enforces them.
- Keep docs and comments aligned with the current `fast-volume-syncer` copy/sync/select/start/stop model.

## Avoid

- Copied references to unrelated project runtimes, manifests, guest consoles, or service APIs.
- TODO comments without an owner or explicit follow-up request.
- Broad style advice that belongs in `docs/guidelines/` rather than code comments.
