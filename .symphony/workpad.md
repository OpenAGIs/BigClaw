# BIG-GO-1018

## Plan
- Migrate the next scoped residual `tests/**` tranche for the frozen legacy Python risk surface.
- Remove `tests/test_risk.py` as stale residual coverage for legacy Python `Scheduler` behavior that is explicitly shimmed to Go mainline replacements, and validate against the current Go risk and worker tests.
- Remove the migrated Python test file from `tests/`.
- Run targeted Go tests for `bigclaw-go/internal/risk` and `bigclaw-go/internal/worker`, capture exact commands and results, then commit and push the branch.

## Acceptance
- Changes stay scoped to this issue's residual `tests/**` tranche.
- The selected Python test behaviors are covered by Go tests against repository code, not tracker metadata.
- The number of repository `.py` files decreases.
- Final report includes impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

## Validation
- `go test ./internal/risk`
- `go test ./internal/worker`
- `find . -name '*.py' | wc -l`
- `find . -name '*.go' | wc -l`
- `git status --short`

## Results
- `cd bigclaw-go && go test ./internal/risk` -> `ok  	bigclaw-go/internal/risk	(cached)`
- `cd bigclaw-go && go test ./internal/worker` -> `ok  	bigclaw-go/internal/worker	1.213s`
- `find . -name '*.py' | wc -l` -> `83`
- `find . -name '*.go' | wc -l` -> `268`
- `git status --short` -> `.symphony/workpad.md`, `bigclaw-go/internal/worker/runtime_test.go` modified; `tests/test_risk.py` deleted
