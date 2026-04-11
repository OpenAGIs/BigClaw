# BIG-GO-215 Python Asset Sweep

## Scope

`BIG-GO-215` (`Residual tooling Python sweep Q`) hardens the already-migrated
tooling, build-helper, and dev-utility surface that used to rely on root Python
wrappers. The active guardrail scope in this lane is `.github`, `.githooks`,
`scripts`, and `bigclaw-go/scripts`.

This checkout already has a repository-wide Python file count of `0`, so the
lane lands as regression prevention and evidence capture rather than an in-branch
Python-file deletion batch.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `.github`: `0` Python files
- `.githooks`: `0` Python files
- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

Explicit remaining Python asset list: none.

The migration and operator guidance for the retired tooling surface remains
documented in `docs/go-cli-script-migration-plan.md` and `README.md`.

## Native Replacement Paths

The Go-first tooling and compatibility entrypoints covering this sweep remain:

- `.github/workflows/ci.yml`
- `.githooks/post-commit`
- `.githooks/post-rewrite`
- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawd/main.go`
- `bigclaw-go/internal/bootstrap/bootstrap.go`
- `bigclaw-go/internal/githubsync/sync.go`
- `bigclaw-go/scripts/benchmark/run_suite.sh`
- `bigclaw-go/scripts/e2e/run_all.sh`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find .github .githooks scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the tooling/build-helper/dev-utility directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO215(RepositoryHasNoPythonFiles|ToolingDirectoriesStayPythonFree|NativeToolingReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.199s`
