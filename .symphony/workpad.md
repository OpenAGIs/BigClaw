# BIG-GO-1026 Workpad

## Plan
- Tighten the existing Go-native coverage for checklist traceability, decision follow-up, and role coverage to fully replace the remaining standalone Python assertions.
- Reuse the existing Go renderers and extend only the test assertions needed to cover the Python contract.
- Reuse the existing Go-native `BuildBIG4204ReviewPack` fixture.
- Remove only the matching Python traceability/role-coverage test from `tests/test_reports.py` after the Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/uigovernance`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the next `UIReviewPack` traceability/role-coverage rendering tranche currently covered by the matching Python tests in `tests/test_reports.py`.
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
- `gofmt -w bigclaw-go/internal/uigovernance/uigovernance_test.go` -> completed
- `go test ./internal/uigovernance` (run from `bigclaw-go/`) -> `ok  	bigclaw-go/internal/uigovernance	0.438s`
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q` -> `71 passed in 0.30s`
- `wc -l tests/test_reports.py` -> `3190 tests/test_reports.py`
- `git diff --stat -- .symphony/workpad.md bigclaw-go/internal/uigovernance/uigovernance_test.go tests/test_reports.py` -> `.symphony/workpad.md | 17 +++++-----------, bigclaw-go/internal/uigovernance/uigovernance_test.go | 8 ++++++++, tests/test_reports.py | 23 ----------------------`
- `rg --files | rg '\\.py$' | wc -l` -> `51`
- `rg --files | rg '\\.go$' | wc -l` -> `288`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$' || true` -> no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` files were touched in this change
