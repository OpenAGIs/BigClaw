# BIG-GO-1573 Workpad

## Plan

1. Confirm the current on-disk state for the `BIG-GO-1573` candidate Python
   files and identify the existing Go replacements or regression coverage.
2. Add a lane-specific regression/doc sweep that keeps the listed retired
   Python assets absent and records the supported Go ownership paths.
3. Record the exact covered file list, validation commands, outcomes, and
   residual risk in repo-native artifacts.
4. Run the targeted validation commands, then commit and push `BIG-GO-1573`.

## Acceptance

- The `BIG-GO-1573` sweep explicitly covers:
  - `src/bigclaw/audit_events.py`
  - `src/bigclaw/execution_contract.py`
  - `src/bigclaw/parallel_refill.py`
  - `src/bigclaw/repo_registry.py`
  - `src/bigclaw/ui_review.py`
  - `tests/test_control_center.py`
  - `tests/test_followup_digests.py`
  - `tests/test_operations.py`
  - `tests/test_repo_governance.py`
  - `tests/test_saved_views.py`
  - `scripts/dev_smoke.py`
  - `bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`
  - `bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`
- Each covered Python path is either deleted already with an explicit Go owner
  recorded, or reduced to a compatibility shim with a deletion condition.
- Validation commands and exact outcomes are captured for the scoped sweep.
- Changes stay scoped to residual Python sweep hardening for this issue.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find . -type f \\( -path './src/bigclaw/audit_events.py' -o -path './src/bigclaw/execution_contract.py' -o -path './src/bigclaw/parallel_refill.py' -o -path './src/bigclaw/repo_registry.py' -o -path './src/bigclaw/ui_review.py' -o -path './tests/test_control_center.py' -o -path './tests/test_followup_digests.py' -o -path './tests/test_operations.py' -o -path './tests/test_repo_governance.py' -o -path './tests/test_saved_views.py' -o -path './scripts/dev_smoke.py' -o -path './bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py' -o -path './bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py' \\) | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1573'`

## Notes

- Current baseline already has the listed candidate Python files deleted, so
  this issue is expected to land as a regression/documentation hardening sweep
  rather than a runtime migration patch.
