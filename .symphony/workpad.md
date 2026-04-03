# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/uigovernance` with the remaining `UIReviewPack` export tranche needed to retire `test_render_ui_review_html_and_bundle_export`.
- Add a Go-native `RenderUIReviewPackHTML` that emits the same section headings exercised by the remaining Python export test and embeds the existing Go report sections.
- Add a Go-native `WriteUIReviewPackBundle` plus `UIReviewPackArtifacts` path contract so the BIG-4204 review pack can emit markdown, HTML, and per-board markdown artifacts from Go.
- Reuse `BuildBIG4204ReviewPack` and add a Go test that mirrors the retired Python HTML/bundle export assertions against the generated files and section renderers.
- Remove only the matching Python UI review HTML/bundle export test from `tests/test_reports.py` after Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/uigovernance`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the remaining `UIReviewPack` HTML/bundle export coverage currently exercised by `test_render_ui_review_html_and_bundle_export` in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/uigovernance` becomes the source of truth for that export contract.
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
  `67 passed in 0.21s`
- `go test ./internal/uigovernance` (run from `bigclaw-go/`)
  `ok  	bigclaw-go/internal/uigovernance	1.812s`
- `wc -l tests/test_reports.py`
  `2762 tests/test_reports.py`
- `git diff --stat`
  `.symphony/workpad.md | 14 +-`
  `bigclaw-go/internal/uigovernance/uigovernance.go | 189 +++++++++++++++++`
  `bigclaw-go/internal/uigovernance/uigovernance_test.go | 186 ++++++++++++++++`
  `tests/test_reports.py | 233 ---------------------`
  `4 files changed, 382 insertions(+), 240 deletions(-)`
- `rg --files | rg '\.py$' | wc -l`
  `51`
- `rg --files | rg '\.go$' | wc -l`
  `288`
- `rg --files | rg '(^|/)(pyproject\.toml|setup\.py|setup\.cfg)$'`
  no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` paths were added or changed in this workspace slice.
