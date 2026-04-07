# BIG-GO-1572 Workpad

## Scope

Sweep the issue's candidate residual Python files and prefer:
- deletion when the file is obsolete,
- replacement with an existing or new Go command/path,
- or reduction to a thin compatibility shim with explicit removal conditions.

Candidate files from the issue:
- `src/bigclaw/__main__.py`
- `src/bigclaw/event_bus.py`
- `src/bigclaw/orchestration.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/service.py`
- `tests/test_console_ia.py`
- `tests/test_execution_flow.py`
- `tests/test_observability.py`
- `tests/test_repo_gateway.py`
- `tests/test_runtime_matrix.py`
- `scripts/create_issues.py`
- `bigclaw-go/scripts/benchmark/soak_local.py`
- `bigclaw-go/scripts/e2e/run_task_smoke.py`

## Plan

1. Inventory the candidate files in the current tree and map each one to its active replacement path, remaining callers, and deletion feasibility.
2. Implement the sweep with the smallest viable change set:
   - delete dead Python files,
   - move runtime behavior to Go-native commands where needed,
   - keep only thin shims where consumers still exist.
3. Update documentation or inline deletion conditions for any shim that must remain.
4. Run targeted validation commands covering the touched replacement paths and record exact results.
5. Commit the change set on `BIG-GO-1572` and push to `origin/BIG-GO-1572`.

## Acceptance

- Produce an explicit list of the Python files covered by this sweep and their final disposition.
- Prefer deletion or Go replacement over retaining Python.
- Any remaining Python file must be a thin compatibility layer with a clear removal condition.
- Provide exact validation commands and outcomes.
- Call out residual risks only where behavior could not be fully exercised.

## Validation

Issue-scoped commands:
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1572'`
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl -run 'TestAutomationUsageListsBIGGO1160GoReplacements|TestAutomationSubscriberTakeoverFaultMatrixBuildsReport|TestRunCreateIssuesCreatesOnlyMissing'`

Expected evidence:
- The Python file inventory should be empty for the full repository.
- The BIG-GO-1572 regression should verify the exact candidate ledger and report content.
- The bigclawctl test slice should verify the touched Go replacement command surfaces.
