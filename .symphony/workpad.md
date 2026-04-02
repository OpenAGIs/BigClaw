# BIG-GO-1099

## Plan

- retire the dead Python wrapper entrypoints `src/bigclaw/__main__.py` and
  `src/bigclaw/legacy_shim.py`
- update Go-side compile-check and regression coverage so the repo documents the
  wrappers as retired instead of frozen shims
- refresh Go-mainline migration docs and README language that still reference
  the removed Python wrappers
- run targeted validation for the changed Go packages and capture exact `.py`
  count reduction evidence
- commit and push the issue branch

## Acceptance

- the tracked repository `.py` count decreases from the pre-change baseline
- `src/bigclaw/__main__.py` and `src/bigclaw/legacy_shim.py` are removed from
  the repo
- active code, tests, and docs no longer describe those two files as retained
  compatibility shims
- targeted Go validation covering `legacy-python` and regression guardrails
  passes

## Validation

- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`
- `git ls-tree -r --name-only HEAD | rg '\.py$' | wc -l`
- `find src -name '*.py' | wc -l`
