# BIG-GO-923

## Plan

1. Inventory the current Python/pytest harness surface centered on `tests/conftest.py`, `tests/`, and any existing Go migration helpers or reports.
2. Identify the smallest missing Go-native harness/reporting pieces needed to satisfy the issue without broad unrelated migration.
3. Implement the first batch of scoped Go changes for harness replacement or migration reporting.
4. Refresh checked-in migration artifacts if code changes affect the current snapshot or migration narrative.
5. Run targeted Go tests and any required generation commands, record exact commands and outcomes, then commit and push the branch.

## Acceptance

- Current Python and non-Go pytest harness assets are explicitly inventoried.
- A Go-native replacement and/or concrete migration path exists for the current `tests/conftest.py` bootstrap behavior.
- First-batch scoped Go implementation is landed in-repo.
- Conditions for deleting legacy Python harness assets are documented and machine-checkable where practical.
- Regression validation commands and their exact results are recorded for this issue.

## Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl legacy-python pytest --repo .. --python python3 -- -- tests/test_planning.py tests/test_audit_events.py -q`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...`

## Validation Results

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && PYTHONPATH=src python3 -m pytest tests/test_planning.py -q`
  Result: passed (`14 passed`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && PYTHONPATH=src python3 -m pytest tests/test_planning.py tests/test_audit_events.py -q`
  Result: passed (`19 passed in 0.08s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl legacy-python pytest --repo .. --python python3 -- -- tests/test_planning.py tests/test_audit_events.py -q`
  Result: passed (`19 passed in 0.06s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/testharness	1.399s`; `ok  	bigclaw-go/internal/regression	1.746s`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --json`
  Result: passed (`status=ok`; `inventory_summary=tests=28 bigclaw_imports=28 pytest_imports=0 pytest_command_refs=0`; `pyproject_exists=true`; `pyproject_declares_pytest=false`; `pyproject_has_pytest_config=false`; `conftest_exists=false`; `conftest_uses_pytest_plugins=false`; `conftest_delete_status.can_delete=true`; `legacy_pytest_delete_status.can_delete=false`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
  Result: passed and refreshed `bigclaw-go/docs/reports/pytest-harness-status.json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/internal/regression	0.877s`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...`
  Result: passed

## Notes

- Keep scope constrained to the pytest/conftest harness migration surface for this issue.
- Do not remove legacy Python assets unless the checked migration gate says they are delete-ready and the replacement coverage is in place.
