# BIG-GO-966 Material Pass

## Lane Inventory

- `tests/test_governance.py`: delete. Replaced by `bigclaw-go/internal/governance/freeze_test.go`.
- `tests/test_reports.py`: keep. Current Go `internal/reporting` coverage does not yet replace the broader report-studio, launch-checklist, and delivery-checklist contracts in this Python file.
- `tests/test_risk.py`: delete. Replaced by `bigclaw-go/internal/risk/risk_test.go` and `bigclaw-go/internal/risk/assessment_test.go`.
- `tests/test_observability.py`: delete. Replaced by `bigclaw-go/internal/observability/recorder_test.go`, `audit_test.go`, and `audit_spec_test.go`.
- `tests/test_planning.py`: delete. Replaced in this issue by `bigclaw-go/internal/planning/planning_test.go`.
- `tests/test_mapping.py`: delete. Replaced by `bigclaw-go/internal/intake/mapping_test.go`.
- `tests/test_memory.py`: delete. Replaced in this issue by `bigclaw-go/internal/memory/store_test.go`.
- `tests/test_operations.py`: delete. Replaced in this issue by `bigclaw-go/internal/reporting/operations_parity_test.go`.
- `tests/test_repo_board.py`: delete. Replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_collaboration.py`: delete. Replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_gateway.py`: delete. Replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_governance.py`: delete. Replaced by `bigclaw-go/internal/repo/governance_test.go`.
- `tests/test_repo_links.py`: delete. Replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_registry.py`: delete. Replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.
- `tests/test_repo_rollout.py`: delete. Replaced in this issue by `bigclaw-go/internal/planning/planning_test.go`.
- `tests/test_repo_triage.py`: delete. Replaced by `bigclaw-go/internal/repo/repo_surfaces_test.go`.

## Result

- Targeted lane Python tests before: `16`
- Targeted lane Python tests after: `1`
- Deleted lane Python tests: `15`
- Repository-wide Python files before: `123`
- Repository-wide Python files after: `108`
- Repository-wide delta: `-15`

## Validation

- `cd bigclaw-go && go test ./internal/governance ./internal/reporting ./internal/risk ./internal/observability ./internal/repo ./internal/intake ./internal/memory`
  - `ok  	bigclaw-go/internal/governance	1.173s`
  - `ok  	bigclaw-go/internal/reporting	1.481s`
  - `ok  	bigclaw-go/internal/risk	1.967s`
  - `ok  	bigclaw-go/internal/observability	2.850s`
  - `ok  	bigclaw-go/internal/repo	3.289s`
  - `ok  	bigclaw-go/internal/intake	3.748s`
  - `ok  	bigclaw-go/internal/memory	2.412s`
- `cd bigclaw-go && go test ./internal/planning`
  - `ok  	bigclaw-go/internal/planning	3.164s`
- `cd bigclaw-go && go test ./internal/reporting`
  - `ok  	bigclaw-go/internal/reporting	3.149s`
