# BIG-GO-923 Workpad

## Plan

1. Audit the current `bigclawctl` migration/legacy command surface and identify a stable Go entrypoint for pytest-harness migration status.
2. Refactor `bigclaw-go/internal/testharness` so pytest-asset inventory can be consumed outside `testing.TB`, while preserving the existing test helpers and assertions.
3. Add a `bigclawctl` command that emits the current pytest-harness inventory and `tests/conftest.py` deletion status in text/JSON form.
4. Extend command and harness tests, update the migration report with the new Go-owned status surface, then run targeted validation, commit, and push.

## Acceptance

- The repository explicitly lists the current pytest harness assets and what `tests/conftest.py` still does.
- `bigclaw-go/internal/testharness` keeps the Go-native replacement helpers for repo/project/src bootstrap and machine-checks the remaining pytest surface without missing direct `pytest` imports.
- `bigclawctl` exposes a stable Go-owned command that reports the pytest harness inventory and current `tests/conftest.py` deletion gate.
- The migration report states when `tests/conftest.py` can be deleted and which regression commands gate that removal.
- The current `tests/conftest.py` deletion blockers remain machine-checked from Go rather than only described in prose.
- The current `tests/conftest.py` delete-readiness summary is available as one stable line from Go-owned harness code.
- The final result includes the exact validation commands executed and whether they passed.

## Validation

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && python3 -m pytest tests/test_mapping.py -q`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./cmd/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --json`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && git status --short`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && git add . && git commit -m "..." && git push origin BIG-GO-923-go-test-harness`

## Validation Results

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923 && python3 -m pytest tests/test_mapping.py -q`
  Result: passed (`.. [100%]`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./internal/testharness`
  Result: passed (`ok  	bigclaw-go/internal/testharness	1.312s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go test ./cmd/bigclawctl`
  Result: passed (`ok  	bigclaw-go/cmd/bigclawctl	2.468s`)
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-923/bigclaw-go && go run ./cmd/bigclawctl pytest-harness --project-root .. --json`
  Result: passed (`status=ok`; `inventory_summary=tests=56 bigclaw_imports=47 pytest_imports=3`; `conftest_delete_status.can_delete=false`)

## Current Status

- `tests/conftest.py` delete-readiness: `conftest_delete_ready=false blockers=56 legacy pytest modules remain under tests/; 47 legacy pytest modules still import bigclaw from src/; 3 legacy pytest modules still import pytest directly`
- Structured delete-readiness status:
  `{"can_delete":false,"legacy_test_modules":56,"bigclaw_import_modules":47,"pytest_import_modules":3}`
