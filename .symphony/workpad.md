# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the remaining checklist and issue-closure reporting contracts still covered only in Python.
- Port the launch checklist, final delivery checklist, documentation artifact, and issue closure decision/evaluation logic into Go without broadening into scheduler or other domains.
- Add focused Go coverage for the matching checklist and issue-closure tests in `tests/test_reports.py`.
- Remove only the matching Python checklist and issue-closure tests from `tests/test_reports.py`.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the remaining checklist and issue-closure reporting contracts currently exercised by the matching `tests/test_reports.py` cases.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for those checklist and issue-closure contracts.
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
  `33 passed in 0.13s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	0.945s`
- `wc -l tests/test_reports.py`
  `1532 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 27 +--\n  bigclaw-go/internal/reporting/reporting.go | 282 ++++++++++++++++++++++++\n  bigclaw-go/internal/reporting/reporting_test.go | 248 +++++++++++++++++++++\n  tests/test_reports.py | 208 -----------------\n  4 files changed, 537 insertions(+), 228 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
