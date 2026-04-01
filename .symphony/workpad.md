# BIG-GO-1026 Workpad

## Plan
- Extend `bigclaw-go/internal/uigovernance` with the next `UIReviewPack` auditor tranche for unresolved decisions plus blocker, freeze-exception, and timeline validation.
- Add typed blocker and blocker-event support in the Go review-pack model where needed by those audit scenarios.
- Reuse and extend the Go-native `BuildBIG4204ReviewPack` fixture to cover the remaining adjacent blocker-oriented audit cases.
- Remove only the matching Python `UIReviewPack` audit tests from `tests/test_reports.py` after the Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/uigovernance`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the next blocker-oriented `UIReviewPack` audit tranche currently covered by the matching Python tests in `tests/test_reports.py`.
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
- `gofmt -w bigclaw-go/internal/uigovernance/uigovernance.go bigclaw-go/internal/uigovernance/uigovernance_test.go` -> completed
- `go test ./internal/uigovernance` (run from `bigclaw-go/`) -> `ok  	bigclaw-go/internal/uigovernance	1.039s`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q` -> `79 passed in 0.32s`
- `wc -l tests/test_reports.py` -> `3424 tests/test_reports.py`
- `git diff --stat` -> `.symphony/workpad.md | 19 +-, bigclaw-go/internal/uigovernance/uigovernance.go | 457 +++++++++++++++++----, bigclaw-go/internal/uigovernance/uigovernance_test.go | 164 ++++++++, tests/test_reports.py | 160 --------`
- `git status --short` -> `M .symphony/workpad.md`, `M bigclaw-go/internal/uigovernance/uigovernance.go`, `M bigclaw-go/internal/uigovernance/uigovernance_test.go`, `M tests/test_reports.py`
- `rg --files | rg '\\.py$' | wc -l` -> `51`
- `rg --files | rg '\\.go$' | wc -l` -> `288`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$' || true` -> no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` files were touched in this change
