# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/reporting` with the shared-view collaboration annotation contract still covered only in Python.
- Add a minimal Go collaboration thread shape to `SharedViewContext` and render collaboration notes/decisions in `renderSharedViewContext`.
- Add focused Go coverage for the shared-view collaboration case exercised by `test_render_shared_view_context_includes_collaboration_annotations`.
- Remove only the matching Python collaboration-annotation test from `tests/test_reports.py` after Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/reporting`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the shared-view collaboration annotation contract currently exercised by `test_render_shared_view_context_includes_collaboration_annotations` in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/reporting` becomes the source of truth for that shared-view collaboration contract.
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
  `61 passed in 0.18s`
- `go test ./internal/reporting` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/reporting	1.164s`
- `wc -l tests/test_reports.py`
  `2639 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 13 ++++---`
  `bigclaw-go/internal/reporting/reporting.go | 50 +++++++++++++++++++++----`
  `bigclaw-go/internal/reporting/reporting_test.go | 39 +++++++++++++++++++`
  `tests/test_reports.py | 38 -------------------`
  `4 files changed, 89 insertions(+), 51 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
