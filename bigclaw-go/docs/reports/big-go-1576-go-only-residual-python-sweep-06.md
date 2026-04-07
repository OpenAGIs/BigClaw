# BIG-GO-1576 Go-only residual Python sweep 06

## Scope

This sweep covers the following residual Python candidates from the issue brief. All of them are
already physically absent in the current branch baseline, so the lane focuses on pinning that state
with an explicit ledger and regression coverage instead of introducing another compatibility shim.

| Candidate path | Sweep status | Go or repo-native replacement |
| --- | --- | --- |
| `src/bigclaw/console_ia.py` | absent | `bigclaw-go/internal/consoleia/consoleia.go`, `bigclaw-go/internal/consoleia/consoleia_test.go` |
| `src/bigclaw/issue_archive.py` | absent | `bigclaw-go/internal/issuearchive/archive.go`, `bigclaw-go/internal/issuearchive/archive_test.go` |
| `src/bigclaw/queue.py` | absent | `bigclaw-go/internal/queue/queue.go`, `bigclaw-go/internal/queue/sqlite_queue_test.go` |
| `src/bigclaw/risk.py` | absent | `bigclaw-go/internal/risk/risk.go`, `bigclaw-go/internal/risk/risk_test.go` |
| `src/bigclaw/workspace_bootstrap.py` | absent | `bigclaw-go/internal/bootstrap/bootstrap.go`, `scripts/ops/bigclawctl` |
| `tests/test_dashboard_run_contract.py` | absent | `bigclaw-go/internal/product/dashboard_run_contract_test.go`, `bigclaw-go/internal/api/expansion_test.go` |
| `tests/test_issue_archive.py` | absent | `bigclaw-go/internal/issuearchive/archive_test.go` |
| `tests/test_parallel_validation_bundle.py` | absent | `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go`, `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go` |
| `tests/test_repo_rollout.py` | absent | `bigclaw-go/internal/product/clawhost_rollout_test.go`, `bigclaw-go/internal/pilot/rollout_test.go` |
| `tests/test_shadow_matrix_corpus.py` | absent | `bigclaw-go/internal/regression/production_corpus_surface_test.go`, `bigclaw-go/internal/regression/python_lane8_remaining_tests_test.go` |
| `scripts/ops/bigclaw_workspace_bootstrap.py` | absent | `scripts/ops/bigclawctl`, `bigclaw-go/internal/bootstrap/bootstrap.go` |
| `bigclaw-go/scripts/e2e/export_validation_bundle.py` | absent | `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`, `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go` |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` | absent | `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands.go`, `bigclaw-go/cmd/bigclawctl/automation_e2e_bundle_commands_test.go` |

## Compatibility posture

- No Python compatibility shim remains for this sweep set.
- Removal condition is already satisfied for every listed asset because the repository baseline is
  physically Python-free and the surviving entrypoints are Go or shell wrappers over Go commands.

## Validation commands

- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1576'`
  Result: `ok  	bigclaw-go/internal/regression	0.374s`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestTopLevelModulePurgeTranche(1|8|14|17)|TestPythonTestTranche17Removed|TestRootScriptResidualSweep|TestE2EScriptDirectoryStaysPythonFree'`
  Result: `ok  	bigclaw-go/internal/regression	0.169s`
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  Result: no output; repository-wide physical Python file count remains `0`.
- `find src/bigclaw tests scripts/ops bigclaw-go/scripts/e2e -type f -name '*.py' 2>/dev/null | sort`
  Result: no output; the focused residual sweep 06 surface remains Python-free.

## Residual risk

- Historical reports and migration plans still mention some removed Python paths as prior-state evidence.
  Those references are documentation history, not live assets.
- This lane does not alter the historical reports under `reports/`; it only locks the current
  repository state so deleted Python files cannot quietly reappear.
