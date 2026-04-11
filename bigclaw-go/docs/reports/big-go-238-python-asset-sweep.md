# BIG-GO-238 Python Asset Sweep

## Scope

Broad repo reduction lane `BIG-GO-238` records the current Go-only repository
baseline with explicit focus on `src/bigclaw`, `tests`, `scripts`, and
`bigclaw-go/scripts`, plus the active root operator and CI surfaces that now
replace the retired Python helpers.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `src/bigclaw`: `0` Python files
- `tests`: `0` Python files
- `scripts`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

This lane therefore lands as a regression-prevention and evidence-refresh sweep
rather than a direct Python-file deletion batch in this checkout.

## Go Or Native Replacement Paths

The active Go/native replacement surface covering this sweep remains:

- `README.md`
- `workflow.md`
- `.github/workflows/ci.yml`
- `scripts/dev_bootstrap.sh`
- `scripts/ops/bigclawctl`
- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawd/main.go`
- `bigclaw-go/scripts/e2e/run_all.sh`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw tests scripts bigclaw-go/scripts -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the priority residual directories remained Python-free.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO238(RepositoryHasNoPythonFiles|PriorityResidualDirectoriesStayPythonFree|ReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.182s`

## Residual Risk

- This lane documents and hardens a repository state that was already
  Python-free; it does not by itself lower the `.py` count in this checkout.
- The replacement surface is primarily operator and evidence oriented, so
  future Python reintroduction risk is mitigated here through absence checks
  and repo-shape validation rather than through a new feature migration.
