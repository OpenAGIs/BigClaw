# BIG-GO-923 Workpad

## Plan

1. Remove `tests/test_workspace_bootstrap.py`, which is already covered by Go bootstrap tests.
2. Refresh pytest-harness inventory counts, snapshot artifact, and migration report so the `conftest` deletion gate reflects the reduced legacy surface again.
3. Run targeted Python/Go validation for the bootstrap/harness/report gates, then commit and push.

## Acceptance

- The repository explicitly lists the current pytest harness assets and what `tests/conftest.py` still does.
- `bigclaw-go/internal/testharness` keeps the Go-native replacement helpers for repo/project/src bootstrap and machine-checks the remaining pytest surface without missing direct `pytest` imports.
- `bigclawctl` exposes a stable Go-owned command that reports the pytest harness inventory and current `tests/conftest.py` deletion gate.
- A checked-in report artifact captures the current pytest-harness inventory and delete-readiness status in machine-readable form.
- Go regression coverage fails if the checked-in pytest-harness snapshot drifts from the current repository inventory.
- The checked-in snapshot is clone-portable and avoids absolute path drift.
- At least one additional legacy pytest module is retired from `tests/` and replaced by Go coverage in this issue.
- The migration report states when `tests/conftest.py` can be deleted and which regression commands gate that removal.
- The current `tests/conftest.py` deletion blockers remain machine-checked from Go rather than only described in prose.
- The current `tests/conftest.py` delete-readiness summary is available as one stable line from Go-owned harness code.
- The final result includes the exact validation commands executed and whether they passed.

## Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && PYTHONPATH=src python3 -c "from bigclaw.mapping import map_priority; from bigclaw.models import Priority; assert map_priority('P0') == Priority.P0"`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/bootstrap ./internal/testharness ./cmd/bigclawctl ./internal/regression`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && git status --short`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && git add . && git commit -m "..." && git push origin BIG-GO-923-go-test-harness`

## Validation Results

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && PYTHONPATH=src python3 -c "from bigclaw.mapping import map_priority; from bigclaw.models import Priority; assert map_priority('P0') == Priority.P0"`
  Result: passed (exit 0)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/bootstrap ./internal/testharness ./cmd/bigclawctl ./internal/regression`
  Result: passed (`ok  	bigclaw-go/internal/bootstrap	4.746s`; `ok  	bigclaw-go/internal/testharness	1.677s`; `ok  	bigclaw-go/cmd/bigclawctl	4.010s`; `ok  	bigclaw-go/internal/regression	1.769s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --report-path docs/reports/pytest-harness-status.json --json`
  Result: passed (`status=ok`; snapshot uses portable repo-relative paths with `project_root=.` and `conftest_path=tests/conftest.py`; `inventory_summary=tests=38 bigclaw_imports=29 pytest_imports=2`; `conftest_delete_status.can_delete=false`)

## Current Status

- `tests/conftest.py` delete-readiness: `conftest_delete_ready=false blockers=38 legacy pytest modules remain under tests/; 29 legacy pytest modules still import bigclaw from src/; 2 legacy pytest modules still import pytest directly`
- Structured delete-readiness status:
  `{"can_delete":false,"legacy_test_modules":38,"bigclaw_import_modules":29,"pytest_import_modules":2}`
