# BIG-GO-1026 Workpad

## Plan
- Port the remaining Python scheduler/tool-runtime contract slice into a dedicated Go package under `bigclaw-go/internal`, keeping the surface limited to the six `BIG301`-`BIG303` tests.
- Remove the matching Python tests from `tests/test_reports.py` only after the Go-native coverage is in place.
- Re-run the targeted reports pytest file and the Go tests for the new package.
- Capture the updated repo inventory and confirm `pyproject.toml` / `setup.py` / `setup.cfg` remain unchanged.
- Commit and push the follow-up reduction on `BIG-GO-1026`.

## Acceptance
- Scope stays limited to the scheduler/tool-runtime contracts currently covered by the six `BIG301`-`BIG303` Python tests.
- Go-native coverage in the new Go package becomes the source of truth for those contracts.
- `tests/test_reports.py` shrinks while the consolidated suite still passes.
- Report includes `.py` / `.go` file-count impact and confirms whether `pyproject.toml` / `setup.py` / `setup.cfg` changed.

## Validation
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q`
- `go test ./internal/runtimecompat`
- `wc -l tests/test_reports.py`
- `git diff --stat`
- `rg --files | rg '\\.py$' | wc -l`
- `rg --files | rg '\\.go$' | wc -l`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$'`

## Validation Results
- `gofmt -w bigclaw-go/internal/runtimecompat/runtimecompat.go bigclaw-go/internal/runtimecompat/runtimecompat_test.go` -> completed
- `PYTHONPATH=src python3 -m pytest tests/test_reports.py -q` -> `120 passed in 0.38s`
- `go test ./internal/runtimecompat` (run from `bigclaw-go/`) -> `ok  	bigclaw-go/internal/runtimecompat	0.408s`
- `wc -l tests/test_reports.py` -> `5104 tests/test_reports.py`
- `git diff --stat` -> `.symphony/workpad.md | 20 ++++------, tests/test_reports.py | 107 --------------------------------------------------`
- `git status --short` -> `M .symphony/workpad.md`, `M tests/test_reports.py`, `?? bigclaw-go/internal/runtimecompat/`
- `rg --files | rg '\\.py$' | wc -l` -> `51`
- `rg --files | rg '\\.go$' | wc -l` -> `286`
- `rg --files | rg '(^|/)(pyproject\\.toml|setup\\.py|setup\\.cfg)$' || true` -> no matches; no `pyproject.toml`, `setup.py`, or `setup.cfg` files were touched in this change
