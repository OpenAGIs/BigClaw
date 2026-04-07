# BIG-GO-1574 Workpad

## Plan

1. Confirm the BIG-GO-1574 candidate Python file inventory and current
   repository Python baseline.
2. Add a lane-specific report and regression guard for the exact candidate
   ledger and mapped Go/native replacement evidence.
3. Run the targeted validation commands, record exact results, then commit and
   push `BIG-GO-1574`.

## Acceptance

- The BIG-GO-1574 candidate Python file ledger is explicit in repo-native
  artifacts.
- Candidate Python files are deleted, migrated, or replaced by Go/native
  ownership evidence.
- If a candidate could not be deleted, it would be reduced to a thin shim with
  a clear deletion condition. Current expectation: no shim remains because the
  candidate paths are already absent.
- Exact validation commands and outcomes are recorded.
- The change is committed and pushed on `BIG-GO-1574`.

## Validation

- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `for f in src/bigclaw/collaboration.py src/bigclaw/github_sync.py src/bigclaw/pilot.py src/bigclaw/repo_triage.py src/bigclaw/validation_policy.py tests/test_cost_control.py tests/test_github_sync.py tests/test_orchestration.py tests/test_repo_links.py tests/test_scheduler.py scripts/ops/bigclaw_github_sync.py bigclaw-go/scripts/e2e/broker_failover_stub_matrix_test.py bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py; do test ! -e "$f" || echo "$f"; done`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1574ResidualPythonSweep04'`

## Notes

- Baseline discovery shows the candidate Python files are already absent on the
  checked-out repository snapshot, so this lane hardens deletion evidence
  rather than reducing a non-zero Python file count.
