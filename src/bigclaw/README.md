# Frozen Legacy Compatibility Tree

`src/bigclaw` is no longer the implementation mainline for BigClaw.

Use `bigclaw-go` for all active runtime, control-plane, tooling, and validation
work. The files in this directory remain only as frozen migration-reference or
compatibility surfaces while the final retirement slices complete.

Allowed changes in this tree are intentionally narrow:

- add or tighten deprecation messaging that points to the Go replacement
- preserve import/runtime compatibility for already-frozen shim entrypoints
- remove modules after the Go replacement and evidence are merged

Do not add new product behavior, validation paths, or operator workflows here.
