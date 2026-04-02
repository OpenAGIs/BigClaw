# BIG-GO-1099

## Plan

- inline the legacy observability/task/audit surface from
  `src/bigclaw/observability.py` into `src/bigclaw/design_system.py`
- inline the legacy UI review surface from `src/bigclaw/ui_review.py` into
  `src/bigclaw/design_system.py`
- update internal imports and package exports so the removed modules continue to
  resolve through compatibility shims
- delete `src/bigclaw/observability.py` and `src/bigclaw/ui_review.py`
- run targeted regression checks for the touched Go mirrors and capture exact
  `.py` count evidence
- commit and push the branch

## Acceptance

- tracked repository `.py` count decreases from the pre-change baseline of `7`
- `src/bigclaw/observability.py` is removed and its exported surface remains
  reachable from the package
- `src/bigclaw/ui_review.py` is removed and its exported surface remains
  reachable from the package
- `src/bigclaw/design_system.py` owns the migrated observability and UI review
  legacy Python surface
- package-level compatibility for `bigclaw.observability` and
  `bigclaw.ui_review` remains intact

## Validation

- `go test ./internal/designsystem ./internal/observability ./internal/uireview`
- `git ls-files '*.py' | wc -l`
- `git diff --stat`
