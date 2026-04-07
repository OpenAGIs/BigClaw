# BIG-GO-1575 Python Asset Sweep

## Scope

Refill lane `BIG-GO-1575` records the residual status for the issue's
candidate Python file set:

- `src/bigclaw/connectors.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/workflow.py`
- `tests/test_cross_process_coordination_surface.py`
- `tests/test_governance.py`
- `tests/test_parallel_refill.py`
- `tests/test_repo_registry.py`
- `tests/test_service.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`

The current checkout is already Python-free, so this lane ships exact-ledger
documentation and regression hardening for these retired paths rather than a
new in-branch deletion batch.

## Before And After Counts

- Repository-wide physical Python file count before lane changes: `0`
- Repository-wide physical Python file count after lane changes: `0`
- Candidate file count still present before lane changes: `0`
- Candidate file count still present after lane changes: `0`

## Candidate File Ledger

All covered candidate files are already absent in the current baseline:

- `src/bigclaw/connectors.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/planning.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/workflow.py`
- `tests/test_cross_process_coordination_surface.py`
- `tests/test_governance.py`
- `tests/test_parallel_refill.py`
- `tests/test_repo_registry.py`
- `tests/test_service.py`
- `scripts/ops/bigclaw_refill_queue.py`
- `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`
- `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py`

Deleted files in this lane: `[]`

Compatibility shims left in this lane: `[]`

## Go Or Native Replacement Paths

The active Go/native replacement surface for this candidate set is:

- `bigclaw-go/internal/intake/connector.go`
- `bigclaw-go/internal/governance/freeze.go`
- `bigclaw-go/internal/planning/planning.go`
- `bigclaw-go/internal/reporting/reporting.go`
- `bigclaw-go/internal/workflow/definition.go`
- `bigclaw-go/internal/service/server.go`
- `bigclaw-go/internal/repo/registry.go`
- `bigclaw-go/internal/refill/queue.go`
- `scripts/ops/bigclawctl`
- `bigclaw-go/cmd/bigclawctl/automation_e2e_coordination_surface_command.go`
- `bigclaw-go/internal/api/validation_bundle_continuation_surface.go`

## Removal Conditions

No Python compatibility shims remain for this candidate set, so there are no
lane-specific shim deletion conditions left to satisfy.

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `for f in src/bigclaw/connectors.py src/bigclaw/governance.py src/bigclaw/planning.py src/bigclaw/reports.py src/bigclaw/workflow.py tests/test_cross_process_coordination_surface.py tests/test_governance.py tests/test_parallel_refill.py tests/test_repo_registry.py tests/test_service.py scripts/ops/bigclaw_refill_queue.py bigclaw-go/scripts/e2e/cross_process_coordination_surface.py bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate_test.py; do [ -e "$f" ] && echo "EXISTS $f" || echo "MISSING $f"; done`
  Result: all 13 paths reported `MISSING`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1575(RepositoryHasNoPythonFiles|CandidatePathsRemainAbsent|ReplacementPathsRemainAvailable|LaneReportCapturesCoverage)$'`
  Result: targeted regression guard passed.
- `cd bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/intake ./internal/governance ./internal/planning ./internal/reporting ./internal/workflow ./internal/repo ./internal/refill ./internal/service`
  Result: targeted Go replacement packages passed.

## Residual Risk

- Historical docs and status artifacts may still mention the removed Python
  paths as migration history, but the repository currently contains no physical
  `.py` files.
