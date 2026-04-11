# BIG-GO-264 Python Asset Sweep

## Scope

`BIG-GO-264` is scoped to the residual mixed-workload CLI helper defaults in
`bigclawctl`. The audited live helper surface is the mixed workload matrix
command and its command-level regression coverage.

The repository baseline in this checkout was already physically free of `.py`
files, so this lane removes Python command residue from helper defaults rather
than deleting new Python files.

## Python Baseline

Repository-wide Python file count: `0`.

Audited helper directory state:

- `bigclaw-go/cmd`: `0` Python files
- `bigclaw-go/internal/regression`: `0` Python files

Explicit remaining Python asset list: none.

## Helper Replacements

The retained Go/native helper surface for this lane is:

- `bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command.go`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_mixed_workload_command_test.go`
- `bigclaw-go/internal/regression/big_go_264_zero_python_guard_test.go`

Residual helper defaults removed by this lane:

- `python -c \"print('gpu via ray')\"` -> `echo gpu via ray`
- `python -c \"print('required ray')\"` -> `echo required ray`

## Why This Sweep Is Safe

The mixed-workload route selection is driven by metadata and executor
requirements, not by a Python runtime dependency in the sample entrypoints.
Replacing the two placeholder commands with shell-native `echo` entrypoints
keeps the helper behavior intact while removing the last Python-shaped residue
from this CLI surface.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find bigclaw-go/cmd bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the mixed-workload helper and regression surfaces remained Python-free.
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationMixedWorkloadMatrixBuildsReport|TestDefaultMixedWorkloadTasksUseNoPythonEntrypoints'`
  Result: `ok  	bigclaw-go/cmd/bigclawctl	0.351s`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO264(MixedWorkloadHelperAvoidsPythonEntrypoints|LaneReportCapturesMixedWorkloadHelperSweep)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.177s`
