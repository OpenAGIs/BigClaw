# BIG-GO-941 Workpad

## Plan

1. Inspect the repository root build/config surfaces and identify the files that keep Python packaging active at the repo root.
2. Convert the root entrypoints and documentation to a Go-first build path with no root Python build backend.
3. Remove obsolete root Python packaging files that are no longer needed for Go-only materialization.
4. Run targeted validation commands for the changed surfaces and record exact commands and results.
5. Commit the scoped change set and push the branch to the remote.

## Acceptance

- Define the lane file list for root build/config removal.
- Remove `pyproject.toml` / `setup.py` as root Python build dependencies and replace them with a Go-only root build entrypoint.
- Document the Go replacement path or deletion plan for any removed surface.
- Provide validation commands, exact results, and residual risks.
- Keep the change set scoped to this issue.

## Validation

- `go test ./...` from `bigclaw-go`
- `make test`
- `make build`
- `git status --short`

## Results

- `cd bigclaw-go && go test ./...` -> passed
- `make test` -> passed
- `make build` -> passed
- `bash scripts/dev_bootstrap.sh` -> passed
