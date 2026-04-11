# BIG-GO-1594 Go-Only Sweep Refill

## Scope

Issue `BIG-GO-1594` covers the remaining Python asset slice called out in the
lane brief:

- `src/bigclaw/collaboration.py`
- `src/bigclaw/github_sync.py`
- `src/bigclaw/pilot.py`
- `src/bigclaw/repo_triage.py`
- `src/bigclaw/validation_policy.py`
- `tests/test_cost_control.py`
- `tests/test_github_sync.py`
- `tests/test_orchestration.py`

## Python Inventory

Repository-wide Python file count before lane changes: `0`.

Repository-wide Python file count after lane changes: `0`.

Explicit remaining Python asset list: none.

This lane therefore lands as regression-prevention evidence. The assigned
Python assets are already absent in this checkout, so the repo-visible work is
the added guardrail and issue evidence that preserve the Go-only surface.

## Go-Owned Replacement Surfaces

- `src/bigclaw/collaboration.py` -> `bigclaw-go/internal/collaboration/thread.go`
- `src/bigclaw/github_sync.py` -> `bigclaw-go/internal/githubsync/sync.go`
- `src/bigclaw/pilot.py` -> `bigclaw-go/internal/pilot/report.go`
- `src/bigclaw/repo_triage.py` -> `bigclaw-go/internal/repo/triage.go`
- `src/bigclaw/validation_policy.py` -> `bigclaw-go/internal/policy/validation.go`
- `tests/test_cost_control.py` -> `bigclaw-go/internal/costcontrol/controller_test.go`
- `tests/test_github_sync.py` -> `bigclaw-go/internal/githubsync/sync_test.go`
- `tests/test_orchestration.py` -> `bigclaw-go/internal/workflow/orchestration_test.go`
- Root CLI bootstrap entrypoint: `scripts/ops/bigclawctl`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `for path in src/bigclaw/collaboration.py src/bigclaw/github_sync.py src/bigclaw/pilot.py src/bigclaw/repo_triage.py src/bigclaw/validation_policy.py tests/test_cost_control.py tests/test_github_sync.py tests/test_orchestration.py; do test ! -e "$path" && printf 'absent %s\n' "$path"; done`
  Result: printed `absent ...` for all eight assigned stale Python paths.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1594(RepositoryHasNoPythonFiles|AssignedPythonAssetsStayAbsent|GoReplacementPathsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok   bigclaw-go/internal/regression 3.222s`

Residual risk: this checkout already started with zero physical Python files, so BIG-GO-1594 hardens that baseline rather than lowering the numeric file count further.
