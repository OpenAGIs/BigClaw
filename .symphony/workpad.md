# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/uigovernance` with the next `UIReviewPack` rendering tranche for summary, persona-readiness, and interaction-coverage surfaces.
- Port the Go helpers and renderers needed for `review_summary_board`, `persona_readiness_board`, and `interaction_coverage_board`.
- Reuse the existing Go-native `BuildBIG4204ReviewPack` fixture.
- Remove only the matching Python summary/persona/interaction test from `tests/test_reports.py` after the Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/uigovernance`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the next `UIReviewPack` summary/persona/interaction rendering tranche currently covered by the matching Python tests in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/uigovernance` becomes the source of truth for those review-pack contracts.
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
- pending
