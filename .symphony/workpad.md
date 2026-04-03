# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the standalone repo-sync audit reporting contract still covered only in Python.
- Port the minimal repo-sync audit data types and markdown renderer into Go without pulling in task-run detail or scheduler/runtime behavior.
- Add focused Go coverage for the matching repo-sync audit test in `tests/test_reports.py`.
- Remove only the matching Python repo-sync audit test from `tests/test_reports.py`.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the standalone repo-sync audit reporting contract currently exercised by the matching `tests/test_reports.py` case.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for that repo-sync audit report contract.
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
  `30 passed in 0.12s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	1.180s`
- `wc -l tests/test_reports.py`
  `1445 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 27 ++---\n  bigclaw-go/internal/reporting/reporting.go | 131 ++++++++++++++++++++++++\n  bigclaw-go/internal/reporting/reporting_test.go | 38 +++++++\n  tests/test_reports.py | 31 ------\n  4 files changed, 176 insertions(+), 51 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
