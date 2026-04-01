# BIG-GO-1052

## Plan
- Audit `bigclaw-go/scripts/e2e` and repo references to removed Python e2e helpers.
- Add/adjust Go regression coverage so tranche 1 e2e helpers stay deleted and wrapper/docs/CI stay Go-only.
- Update README/workflow/CI references that still imply Python e2e entrypoints.
- Run targeted validation, record exact commands and outcomes, then commit and push.

## Acceptance
- `bigclaw-go/scripts/e2e` contains no tranche-1 Python helpers and regression coverage fails if they reappear.
- README/workflow/hooks/CI references for the migrated e2e entrypoints point to Go/shell entrypoints only.
- Targeted tests pass and exact commands/results are recorded.
- Changes stay scoped to this issue.

## Validation
- `go test ./cmd/bigclawctl ./internal/regression`
- `go test ./...` only if targeted coverage indicates broader breakage risk.
- `git diff --check`
- `git status --short`

## Results
- Audited `.github/workflows`, repo hooks, and checked-in docs for direct tranche-1 `bigclaw-go/scripts/e2e/*.py` entrypoint usage. No remaining workflow or hook invocations were present; the remaining drift was documentation language and missing regression coverage.
- Added Go regression coverage to fail if tranche-1 e2e Python helpers reappear and to assert `scripts/e2e/run_all.sh` stays wired to Go entrypoints.
- Updated Go-facing README and e2e migration docs to describe `scripts/e2e/` as a Go-and-shell-only surface.

## Validation Results
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression` -> passed
- `git diff --check` -> passed
- `find . -path './.git' -prune -o -name '*.py' -print | wc -l` -> `50`
- Validation report added: `reports/BIG-GO-1052-validation.md`
