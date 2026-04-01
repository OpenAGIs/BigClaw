# BIG-GO-1026 Workpad

## Plan
- Extend the `bigclaw-go/internal/uigovernance` package with the self-contained `ConsoleIA` contracts from `src/bigclaw/console_ia.py`.
- Remove the matching Python `ConsoleIA` tests from `tests/test_reports.py` only after the Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for `./internal/uigovernance`.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the `ConsoleIA` contracts currently covered by the matching Python tests in `tests/test_reports.py`.
- Go-native coverage in `bigclaw-go/internal/uigovernance` becomes the source of truth for those contracts.
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
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q` -> `100 passed in 0.45s`
- `go test ./internal/uigovernance` (run from `bigclaw-go/`) -> `ok  	bigclaw-go/internal/uigovernance	(cached)`
- `wc -l tests/test_reports.py` -> `4223 tests/test_reports.py`
- `git diff --stat` -> `.symphony/workpad.md | 16 +-, bigclaw-go/internal/uigovernance/uigovernance.go | 345 +++++++++++++++++++++, bigclaw-go/internal/uigovernance/uigovernance_test.go | 181 +++++++++++, tests/test_reports.py | 203 ------------`
- `git status --short` -> `M .symphony/workpad.md`, `M bigclaw-go/internal/uigovernance/uigovernance.go`, `M bigclaw-go/internal/uigovernance/uigovernance_test.go`, `M tests/test_reports.py`
- `rg --files | rg '\\.py$' | wc -l` -> `51`
- `rg --files | rg '\\.go$' | wc -l` -> `288`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$' || true` -> no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` files were touched in this change
