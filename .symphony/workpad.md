# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/observability` with the explicit manual-takeover audit-spec validation case still covered only in Python.
- Add focused Go coverage for the missing-required-fields path exercised by `test_task_run_audit_spec_event_requires_required_fields`.
- Remove only the matching Python audit-spec validation test from `tests/test_reports.py` after Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/observability`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the audit-spec validation contract currently exercised by `test_task_run_audit_spec_event_requires_required_fields` in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/observability` becomes the source of truth for that validation contract.
- `tests/test_reports.py` shrinks while the consolidated suite still passes.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `go test ./internal/uigovernance`
- `wc -l tests/test_reports.py`
- `git diff --stat`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`

## Validation Results
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
  `62 passed in 0.18s`
- `go test ./internal/observability` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/observability	0.836s`
- `wc -l tests/test_reports.py`
  `2677 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 13 ++++++-------`
  `bigclaw-go/internal/observability/audit_test.go | 15 +++++++++++++++`
  `tests/test_reports.py | 17 -----------------`
  `3 files changed, 21 insertions(+), 24 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
