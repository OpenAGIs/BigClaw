# BIG-GO-1099

## Plan

- add a regression guard that locks the tracked Python floor at the current
  single-file package-root surface
- assert that `src/bigclaw/__init__.py` is the only tracked `.py` file left in
  the repository
- run targeted regression validation for the new guardrail and record the exact
  count evidence
- commit and push the branch

## Acceptance

- repo regression coverage fails if any additional tracked `.py` file is added
- regression coverage documents that `src/bigclaw/__init__.py` is the only
  remaining tracked Python file
- tracked repository `.py` count remains `1`

## Validation

- `cd bigclaw-go && go test ./internal/regression`
- `git ls-files '*.py' | wc -l`
- `git ls-files '*.py'`
