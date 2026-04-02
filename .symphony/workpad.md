# BIG-GO-1099

## Plan

- remove the final tracked Python file at `src/bigclaw/__init__.py`
- update Go regression coverage that still hard-codes `src/bigclaw/__init__.py`
  as the residual Python surface so the repo-wide floor becomes zero
- validate the legacy shim compile check still short-circuits cleanly with no
  Python files present
- run targeted regression validation and record exact command results
- commit and push the branch

## Acceptance

- `git ls-files '*.py'` returns no tracked Python files
- repo regression coverage fails if any tracked `.py` file is reintroduced
- removal of `src/bigclaw/__init__.py` is covered by targeted Go regression
  tests
- tracked repository `.py` count becomes `0`

## Validation

- `go test ./bigclaw-go/internal/regression ./bigclaw-go/internal/legacyshim`
- `git ls-files '*.py'`
- `git ls-files '*.py' | wc -l`
- `git ls-files '*.py'`
