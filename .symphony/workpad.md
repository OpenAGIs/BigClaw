# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with orchestration flow-collaboration reconstruction from ledger audits.
- Add Go-native collaboration thread helpers and canvas rendering support for collaboration summaries, comments, and decision notes.
- Add focused Go coverage for the remaining `test_orchestration_canvas_reconstructs_flow_collaboration_from_ledger` contract.
- Remove only the matching Python orchestration collaboration test from `tests/test_reports.py`.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the remaining orchestration collaboration contract currently exercised by the matching `tests/test_reports.py` case.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for that collaboration contract.
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
  `48 passed in 0.16s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	1.248s`
- `wc -l tests/test_reports.py`
  `1970 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 11 +-\n  bigclaw-go/internal/reporting/reporting.go | 153 +++++++++++++++++++++---\n  bigclaw-go/internal/reporting/reporting_test.go | 78 ++++++++++++\n  tests/test_reports.py | 68 -----------\n  4 files changed, 223 insertions(+), 87 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
