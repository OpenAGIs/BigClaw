# BIG-GO-194 Workpad

## Plan

1. Inventory Python-named scripts, wrappers, and CLI helpers that already have non-Python behavior or Go-backed replacements.
2. Replace residual Python entrypoints in scope with non-Python paths, keeping compatibility where practical through shell wrappers.
3. Update repo references and packaging/lint configuration so the renamed helpers are the documented canonical surface.
4. Run targeted validation for the touched wrappers and helper commands, then record exact commands and outcomes.
5. Commit the scoped branch changes and push `BIG-GO-194` to `origin`.

## Acceptance Criteria

- Python-named residual scripts/wrappers in the touched ops/helper surface are removed or replaced with non-Python equivalents.
- Canonical documentation and config no longer point at removed Python helper paths in the touched area.
- The changed wrapper/CLI flows still execute through the Go-backed toolchain.
- Validation is recorded with exact commands and concrete pass/fail results.
- The issue branch is committed and pushed.

## Validation

- Search for remaining references to the renamed helper paths.
- Run targeted `go test` coverage for `bigclawctl` command behavior.
- Execute the updated shell helper entrypoints with `--help` or equivalent smoke commands.
- Capture `git status --short` before commit and after validation.

## Validation Results

- `go test ./cmd/bigclawctl` (in `bigclaw-go`) -> `ok  	bigclaw-go/cmd/bigclawctl	0.525s`
- `bash scripts/dev-smoke` -> passed; executed `go test ./...` under `bigclaw-go` with all listed packages passing
- `bash scripts/ops/bigclaw-github-sync status --json` -> exited `0` with `status: "ok"` on branch `BIG-GO-194`
- `bash scripts/ops/bigclaw-refill-queue --local-issues local-issues.json` -> exited `0`; dry-run reported `queue_drained: true` and no candidates
- `bash scripts/ops/bigclaw-workspace-bootstrap --help` -> `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `bash scripts/ops/symphony-workspace-bootstrap --help` -> `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `bash scripts/ops/symphony-workspace-validate --help` -> exited `0` and printed `bigclawctl workspace validate` flag help
- `rg -n 'scripts/ops/(bigclaw-github-sync|bigclaw-refill-queue|bigclaw-workspace-bootstrap|symphony-workspace-bootstrap|symphony-workspace-validate)|scripts/dev-smoke' README.md docs pyproject.toml scripts` -> only the new non-Python helper paths remain in active docs/scripts
