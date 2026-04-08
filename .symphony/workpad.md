# BIG-GO-15

## Plan

1. Inspect the `origin/main` root tooling/workflow surfaces for any remaining Python-centric packaging or repo-hygiene residuals.
2. Remove the in-scope residual config files and update the root CI/readme workflow guidance to use the Go-only entrypoints.
3. Run targeted validation for the edited workflow and root operator commands.
4. Commit the scoped changes and push branch `BIG-GO-15`.

## Acceptance

- Remaining Python-centric root tooling residuals for this batch are removed.
- Root workflow/docs point at the Go-only entrypoints that remain supported.
- Validation commands and exact outcomes are captured.
- Changes stay scoped to `BIG-GO-15`.

## Validation

- `git diff --check`
- `git check-ignore -v .venv/example`
- `make test`
- `make build`
- `bash scripts/ops/bigclawctl github-sync --help >/dev/null`
- `bash scripts/ops/bigclawctl dev-smoke`
