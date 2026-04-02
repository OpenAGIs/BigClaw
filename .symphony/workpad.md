# BIG-GO-1081 Workpad

## Plan

1. Remove the remaining redundant Python workspace bootstrap shim that still keeps a Python bootstrap entrypoint alive.
2. Update operator-facing documentation to point issue/bootstrap workflows at `bash scripts/ops/bigclawctl ...` instead of Python paths.
3. Trim validation/test coverage that still assumes the deleted Python shim exists, then run targeted verification.
4. Commit the scoped change set and push the branch.

## Acceptance

- Repository `.py` file count decreases from the current baseline.
- The deleted Python file is no longer present in the repository.
- Repo docs describe a non-Python issue/bootstrap path using `scripts/ops/bigclawctl`.
- Targeted tests/commands covering the touched bootstrap path succeed.

## Validation

- `rg --files | rg '\.py$' | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl`
- `bash scripts/ops/bigclawctl workspace --help`
- `bash scripts/ops/bigclawctl issue --help`
- `git status --short`

## Results

- Python file count: `23 -> 22`
- `cd bigclaw-go && go test ./internal/legacyshim ./cmd/bigclawctl` -> passed
- `bash scripts/ops/bigclawctl workspace --help` -> `usage: bigclawctl workspace <bootstrap|cleanup|validate> [flags]`
- `bash scripts/ops/bigclawctl issue --help` -> passed
