# BIG-GO-1099

## Plan

- retire the dead Python wrapper entrypoints `src/bigclaw/__main__.py` and
  `src/bigclaw/legacy_shim.py`
- inline the tiny deprecation helper into `src/bigclaw/runtime.py` and delete
  `src/bigclaw/deprecation.py`
- inline the small legacy risk scorer into `src/bigclaw/runtime.py` and delete
  `src/bigclaw/risk.py`
- inline the small audit-event registry into `src/bigclaw/observability.py` and
  delete `src/bigclaw/audit_events.py`
- update Go-side compile-check and regression coverage so the repo documents the
  wrappers as retired instead of frozen shims
- refresh active Go-mainline migration docs and README language that still
  reference the removed Python wrappers or the deleted helper file
- run targeted validation for the changed Go packages and capture exact `.py`
  count reduction evidence
- commit and push the issue branch

## Acceptance

- the tracked repository `.py` count decreases from the pre-change baseline
- `src/bigclaw/__main__.py` and `src/bigclaw/legacy_shim.py` are removed from
  the repo
- `src/bigclaw/deprecation.py` is removed with its helper logic preserved in
  `src/bigclaw/runtime.py`
- `src/bigclaw/risk.py` is removed with its legacy scorer logic preserved in
  `src/bigclaw/runtime.py`
- `src/bigclaw/audit_events.py` is removed with its event-spec logic preserved
  in `src/bigclaw/observability.py`
- active code, tests, and docs no longer describe those two files as retained
  compatibility shims
- targeted Go validation covering `legacy-python` and regression guardrails
  passes

## Validation

- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l`
- `find src -name '*.py' | wc -l`
