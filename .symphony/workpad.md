# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/uigovernance` with the remaining `UIReviewPack` report tranche needed to retire `test_build_big_4204_review_pack_is_ready_for_design_sprint_review`.
- Port the simple Go renderers still missing from the consolidated report: decision log, role matrix, signoff log, blocker log, and blocker timeline.
- Expand `RenderUIReviewPackReport` to compose the existing Go-native review boards and logs in the same order and with the same section headers exercised by the Python report test.
- Reuse the existing Go-native `BuildBIG4204ReviewPack` fixture and add a Go test that mirrors the retired Python report assertions.
- Remove only the matching Python design-sprint-review report test from `tests/test_reports.py` after the Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/uigovernance`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the remaining consolidated `UIReviewPack` report coverage currently exercised by `test_build_big_4204_review_pack_is_ready_for_design_sprint_review` in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/uigovernance` becomes the source of truth for that report contract.
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
  `68 passed in 0.26s`
- `go test ./internal/uigovernance` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/uigovernance	0.881s`
- `wc -l tests/test_reports.py`
  `2995 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 13 +-`
  `bigclaw-go/internal/uigovernance/uigovernance.go | 170 ++++++++++++++++++++-`
  `bigclaw-go/internal/uigovernance/uigovernance_test.go | 134 ++++++++++++++++`
  `tests/test_reports.py | 136 -----------------`
  `4 files changed, 310 insertions(+), 143 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
