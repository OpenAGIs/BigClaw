# BIG-GO-1595 Python Asset Sweep

## Scope

`BIG-GO-1595` is a Go-only sweep refill lane for the already-retired
`src/bigclaw` and root-`tests` Python surface covering connectors, governance,
planning, reports, workflow, and representative coordination/governance/refill
tests. In this checkout those Python files are already gone, so the lane lands
as regression hardening and exact replacement evidence rather than a fresh
deletion batch.

Representative retired Python paths pinned by this lane:

- `src/bigclaw/connectors.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/workflow.py`
- `tests/test_cross_process_coordination_surface.py`
- `tests/test_governance.py`
- `tests/test_parallel_refill.py`

## Python Baseline

Repository-wide Python file count: `0`.

Focused directory state:

- `src/bigclaw`: absent
- `tests`: absent

This checkout therefore stays aligned with the Go-only mainline baseline. The
issue cannot reduce the physical Python count numerically because the assigned
paths were already absent at branch entry.

## Go Or Native Replacement Paths

The active Go-owned replacement surface for this retired slice remains:

- `docs/go-domain-intake-parity-matrix.md`
- `bigclaw-go/internal/intake/connector.go`
- `bigclaw-go/internal/governance/freeze.go`
- `bigclaw-go/internal/planning/planning.go`
- `bigclaw-go/internal/reporting/reporting.go`
- `bigclaw-go/internal/workflow/orchestration.go`
- `bigclaw-go/internal/api/coordination_surface.go`
- `bigclaw-go/internal/refill/queue.go`
- `bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`
- `bigclaw-go/docs/reports/parallel-validation-matrix.md`

These paths keep the migrated connector intake, governance, planning, reporting,
workflow, coordination, and refill contracts on Go-owned surfaces.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `find src/bigclaw tests -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the assigned `src/bigclaw` and `tests` Python surface
  remained absent.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1595(RepositoryHasNoPythonFiles|AssignedPythonSourceAndTestsRemainAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.187s`

## Residual Risk

- The lane hardens an already-zero baseline rather than deleting live Python
  files in-branch.
- Future drift risk sits in the Go replacement surfaces above, especially if
  their coverage changes without updating the zero-Python regression lane.
