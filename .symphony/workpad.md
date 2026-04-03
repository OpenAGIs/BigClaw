# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the orchestration portfolio and billing-entitlements rollup/rendering behavior still covered only in Python.
- Add Go-native portfolio/page aggregates plus markdown and HTML renderers that match the current Python reporting contracts.
- Add focused Go coverage for orchestration portfolio rollups, shared-view empty state, overview HTML, billing-entitlements rollups, and ledger extraction.
- Remove only the matching orchestration portfolio and billing-entitlements Python tests from `tests/test_reports.py` after Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the orchestration portfolio and billing-entitlements contracts currently exercised by the corresponding `tests/test_reports.py` cases.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for those portfolio/page contracts.
- `tests/test_reports.py` shrinks while the consolidated suite still passes.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `go test ./internal/reporting`
- `wc -l tests/test_reports.py`
- `git diff --stat`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`

## Validation Results
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
  `51 passed in 0.16s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	0.875s`
- `wc -l tests/test_reports.py`
  `2181 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 12 +-\n  bigclaw-go/internal/reporting/reporting.go | 429 ++++++++++++++++++++++--\n  bigclaw-go/internal/reporting/reporting_test.go | 350 +++++++++++++++++++\n  tests/test_reports.py | 323 ------------------\n  4 files changed, 766 insertions(+), 348 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
