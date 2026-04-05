# BIG-GO-1467 Python Bootstrap Surface Sweep

Date: 2026-04-06

## Scope

Issue: `BIG-GO-1467`

Title: `Lane refill: eliminate remaining Python bootstrap/workspace helper surfaces and validation hooks`

This lane confirmed the checkout was already free of physical `.py` files, then
removed the remaining Python-adjacent bootstrap residue that still shipped in
the repository as non-`.py` assets or references.

## Inventory

- Repository-wide Python file count: `0`.
- `src/bigclaw`: `0` Python files.
- `tests`: `0` Python files.
- `scripts`: `0` Python files.
- `bigclaw-go/scripts`: `0` Python files.

## Deleted Or Replaced Surfaces

- Deleted root Python validation hook config: `.pre-commit-config.yaml`.
  Delete condition: the repository no longer ships Python files, and the active
  repo hygiene path is the Go/bootstrap smoke plus direct Python-inventory
  checks.
- Removed Python bootstrap template references: `workspace_bootstrap.py`, `workspace_bootstrap_cli.py`.
  Go replacement: `scripts/ops/bigclawctl`, `scripts/dev_bootstrap.sh`, and
  `bigclaw-go/internal/bootstrap/bootstrap.go`.
- Replaced README repo-hygiene guidance that invoked `pre-commit run --all-files`.
  Go/native replacement: `bash scripts/dev_bootstrap.sh` and direct inventory
  validation with `find`.

## Active Go / Native Replacements

- Root operator entrypoint: `scripts/ops/bigclawctl`
- Root bootstrap smoke: `scripts/dev_bootstrap.sh`
- Go bootstrap implementation: `bigclaw-go/internal/bootstrap/bootstrap.go`
- Go bootstrap tests: `bigclaw-go/internal/bootstrap/bootstrap_test.go`
- Go CLI entrypoint: `bigclaw-go/cmd/bigclawctl/main.go`
- Lane regression guard: `bigclaw-go/internal/regression/big_go_1467_zero_python_guard_test.go`

## Validation Commands

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1467(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|GoBootstrapSurfacesRemainWithoutPythonHooks|LaneReportCapturesBootstrapHookRetirement)$'`
- `cd bigclaw-go && go test -count=1 ./internal/bootstrap ./cmd/bigclawctl`
