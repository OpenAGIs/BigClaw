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
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/risk ./internal/triage ./internal/workflow ./internal/billing ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/worker ./internal/workflow ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/scheduler ./internal/worker ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/observability ./internal/worker ./internal/workflow ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/product ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/contract ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/scheduler ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/repo ./internal/api ./internal/testharness ./internal/regression ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/product`
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
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/legacyshim ./internal/testharness ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/legacyshim	3.017s`; `ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/legacyshim ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/legacyshim	(cached)`; `ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/internal/regression	3.640s`; `ok  	bigclaw-go/cmd/bigclawctl	5.195s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/refill ./internal/testharness ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/refill	3.319s`; `ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/refill ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/refill	(cached)`; `ok  	bigclaw-go/internal/testharness	1.551s`; `ok  	bigclaw-go/internal/regression	1.868s`; `ok  	bigclaw-go/cmd/bigclawctl	4.406s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/repo ./internal/testharness ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/repo	1.128s`; `ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/repo ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/repo	(cached)`; `ok  	bigclaw-go/internal/testharness	1.580s`; `ok  	bigclaw-go/internal/regression	1.818s`; `ok  	bigclaw-go/cmd/bigclawctl	3.288s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/product ./internal/testharness ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/product	3.175s`; `ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/product ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/product	(cached)`; `ok  	bigclaw-go/internal/testharness	0.777s`; `ok  	bigclaw-go/internal/regression	0.831s`; `ok  	bigclaw-go/cmd/bigclawctl	2.478s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/scheduler ./internal/testharness ./cmd/bigclawctl`
  Result: scheduler package passed before harness snapshot assertions drifted (`ok  	bigclaw-go/internal/scheduler	1.399s`; `ok  	bigclaw-go/internal/testharness	(cached)`), then `cmd/bigclawctl` failed only because `pytest-harness` expectations still referenced the pre-removal count
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --json`
  Result: passed (`status=ok`; `inventory_summary=tests=17 bigclaw_imports=17 pytest_imports=0 pytest_command_refs=0`; `pyproject_exists=true`; `pyproject_declares_pytest=false`; `pyproject_has_pytest_config=false`; `conftest_exists=false`; `conftest_uses_pytest_plugins=false`; `conftest_delete_status.can_delete=true`; `legacy_pytest_delete_status.can_delete=false`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
  Result: passed and refreshed `bigclaw-go/docs/reports/pytest-harness-status.json` (`status=ok`; `inventory_summary=tests=17 bigclaw_imports=17 pytest_imports=0 pytest_command_refs=0`; `conftest_delete_status.summary=conftest_delete_ready=true blockers=none`; `legacy_pytest_delete_status.summary=legacy_pytest_delete_ready=false blockers=17 legacy pytest modules remain under tests/; 17 legacy pytest modules still import bigclaw from src/`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/scheduler ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/scheduler	(cached)`; `ok  	bigclaw-go/internal/testharness	1.666s`; `ok  	bigclaw-go/internal/regression	2.024s`; `ok  	bigclaw-go/cmd/bigclawctl	3.426s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/risk ./internal/triage ./internal/workflow ./internal/billing ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/risk	(cached)`; `ok  	bigclaw-go/internal/triage	(cached)`; `ok  	bigclaw-go/internal/workflow	(cached)`; `ok  	bigclaw-go/internal/billing	(cached)`; `ok  	bigclaw-go/internal/testharness	1.192s`; `ok  	bigclaw-go/internal/regression	1.443s`; `ok  	bigclaw-go/cmd/bigclawctl	3.493s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/worker ./internal/workflow ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/worker	(cached)`; `ok  	bigclaw-go/internal/workflow	(cached)`; `ok  	bigclaw-go/internal/testharness	1.687s`; `ok  	bigclaw-go/internal/regression	2.054s`; `ok  	bigclaw-go/cmd/bigclawctl	4.913s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/scheduler ./internal/worker ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/scheduler	(cached)`; `ok  	bigclaw-go/internal/worker	(cached)`; `ok  	bigclaw-go/internal/testharness	1.209s`; `ok  	bigclaw-go/internal/regression	1.545s`; `ok  	bigclaw-go/cmd/bigclawctl	3.513s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/observability ./internal/worker ./internal/workflow ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/observability	(cached)`; `ok  	bigclaw-go/internal/worker	(cached)`; `ok  	bigclaw-go/internal/workflow	(cached)`; `ok  	bigclaw-go/internal/testharness	1.555s`; `ok  	bigclaw-go/internal/regression	1.954s`; `ok  	bigclaw-go/cmd/bigclawctl	4.006s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/product ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/product	0.476s`; `ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/internal/regression	1.389s`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...`
  Result: passed (`ok  	bigclaw-go/internal/api	2.662s`; `ok  	bigclaw-go/internal/regression	1.783s`; `ok  	bigclaw-go/internal/product	(cached)`; remaining packages passed cached or had no test files)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/contract ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/contract	(cached)`; `ok  	bigclaw-go/internal/testharness	1.237s`; `ok  	bigclaw-go/internal/regression	1.476s`; `ok  	bigclaw-go/cmd/bigclawctl	3.482s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
  Result: passed and refreshed `bigclaw-go/docs/reports/pytest-harness-status.json` (`status=ok`; `inventory_summary=tests=17 bigclaw_imports=17 pytest_imports=0 pytest_command_refs=0`; `conftest_delete_status.summary=conftest_delete_ready=true blockers=none`; `legacy_pytest_delete_status.summary=legacy_pytest_delete_ready=false blockers=17 legacy pytest modules remain under tests/; 17 legacy pytest modules still import bigclaw from src/`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...`
  Result: passed (`ok  	bigclaw-go/cmd/bigclawctl	(cached)`; `ok  	bigclaw-go/internal/contract	(cached)`; `ok  	bigclaw-go/internal/regression	1.520s`; remaining packages passed cached or had no test files)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/scheduler ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/scheduler	(cached)`; `ok  	bigclaw-go/internal/testharness	1.558s`; `ok  	bigclaw-go/internal/regression	1.812s`; `ok  	bigclaw-go/cmd/bigclawctl	3.888s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
  Result: passed and refreshed `bigclaw-go/docs/reports/pytest-harness-status.json` (`status=ok`; `inventory_summary=tests=16 bigclaw_imports=16 pytest_imports=0 pytest_command_refs=0`; `conftest_delete_status.summary=conftest_delete_ready=true blockers=none`; `legacy_pytest_delete_status.summary=legacy_pytest_delete_ready=false blockers=16 legacy pytest modules remain under tests/; 16 legacy pytest modules still import bigclaw from src/`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...`
  Result: passed (`ok  	bigclaw-go/cmd/bigclawctl	4.278s`; `ok  	bigclaw-go/internal/api	4.087s`; `ok  	bigclaw-go/internal/regression	3.319s`; `ok  	bigclaw-go/internal/testharness	3.678s`; remaining packages passed cached or had no test files)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/repo ./internal/api ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/repo	0.782s`; `ok  	bigclaw-go/internal/api	2.646s`; `ok  	bigclaw-go/internal/testharness	(cached)`; `ok  	bigclaw-go/internal/regression	0.930s`; `ok  	bigclaw-go/cmd/bigclawctl	(cached)`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
  Result: passed and refreshed `bigclaw-go/docs/reports/pytest-harness-status.json` (`status=ok`; `inventory_summary=tests=15 bigclaw_imports=15 pytest_imports=0 pytest_command_refs=0`; `conftest_delete_status.summary=conftest_delete_ready=true blockers=none`; `legacy_pytest_delete_status.summary=legacy_pytest_delete_ready=false blockers=15 legacy pytest modules remain under tests/; 15 legacy pytest modules still import bigclaw from src/`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/repo ./internal/api ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/repo	(cached)`; `ok  	bigclaw-go/internal/api	2.226s`; `ok  	bigclaw-go/internal/testharness	1.626s`; `ok  	bigclaw-go/internal/regression	1.786s`; `ok  	bigclaw-go/cmd/bigclawctl	3.868s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/product`
  Result: passed (`ok  	bigclaw-go/internal/product	1.113s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
  Result: passed and refreshed `bigclaw-go/docs/reports/pytest-harness-status.json` (`status=ok`; `inventory_summary=tests=14 bigclaw_imports=14 pytest_imports=0 pytest_command_refs=0`; `conftest_delete_status.summary=conftest_delete_ready=true blockers=none`; `legacy_pytest_delete_status.summary=legacy_pytest_delete_ready=false blockers=14 legacy pytest modules remain under tests/; 14 legacy pytest modules still import bigclaw from src/`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/product ./internal/testharness ./internal/regression ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/internal/product	(cached)`; `ok  	bigclaw-go/internal/testharness	1.236s`; `ok  	bigclaw-go/internal/regression	1.570s`; `ok  	bigclaw-go/cmd/bigclawctl	2.998s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...`
  Result: passed (`ok  	bigclaw-go/cmd/bigclawctl	3.296s`; `ok  	bigclaw-go/cmd/bigclawd	1.805s`; `ok  	bigclaw-go/internal/api	2.288s`; `ok  	bigclaw-go/internal/regression	1.517s`; remaining packages passed cached or had no test files)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./...`
  Result: passed (`ok  	bigclaw-go/cmd/bigclawctl	3.464s`; `ok  	bigclaw-go/cmd/bigclawd	2.502s`; `ok  	bigclaw-go/internal/api	1.992s`; `ok  	bigclaw-go/internal/regression	0.916s`; remaining packages passed cached or had no test files)

## Notes

- Keep scope constrained to the pytest/conftest harness migration surface for this issue.
- Do not remove legacy Python assets unless the checked migration gate says they are delete-ready and the replacement coverage is in place.
