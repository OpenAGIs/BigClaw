# BIG-GO-251 Python Asset Sweep

## Scope

`BIG-GO-251` (`Residual src/bigclaw Python sweep V`) records the assigned
tranche-12 `src/bigclaw` module slice:

- `src/bigclaw/dsl.py`

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `src/bigclaw`: `0` Python files

Explicit assigned Python asset list:

- `src/bigclaw/dsl.py`

The assigned file was already absent in this checkout, so this lane lands as
regression hardening plus evidence capture rather than a fresh `.py` deletion
batch.

## Go Or Native Replacement Paths

The active Go/native replacement surface covering this tranche is:

- `bigclaw-go/internal/workflow/definition.go`
- `bigclaw-go/internal/workflow/definition_test.go`
- `bigclaw-go/internal/workflow/engine.go`
- `bigclaw-go/internal/regression/top_level_module_purge_tranche12_test.go`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `for path in src/bigclaw/dsl.py; do test ! -e "$path" && printf 'absent %s\n' "$path"; done`
  Result: the assigned tranche-12 path reported `absent`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO251(RepositoryHasNoPythonFiles|SrcBigclawTranche12PathRemainsAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche12$'`
  Result: `ok  	bigclaw-go/internal/regression	1.425s`

## Residual Risk

- The branch baseline was already Python-free, so `BIG-GO-251` cannot reduce
  the physical `.py` file count further in this checkout; it can only preserve
  and document the Go-only state for the assigned tranche.
- `src/bigclaw/dsl.py` is also referenced by broader sweep evidence elsewhere
  in the repo, so this lane adds tranche-specific regression hardening rather
  than uniquely owning that historical deletion.
