# BIG-GO-1098 Workpad

## Plan
- Replace residual Python test references in `bigclaw-go/internal/planning/planning.go` with Go-only validation commands and Go evidence links.
- Update planning unit tests so the backlog contract asserts Go-native replacements instead of deleted `tests/*.py` assets.
- Add regression coverage that blocks reintroduction of removed Python test references inside the Go planning backlog.

## Acceptance
- `bigclaw-go/internal/planning/planning.go` no longer contains `pytest` commands or `tests/test_*.py` evidence targets for the v3 candidate backlog.
- Go tests document the replacement packages that now serve as validation evidence for release control, ops hardening, and orchestration rollout.
- Targeted Go test suites pass.

## Validation
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1098/bigclaw-go && go test ./internal/planning ./internal/regression`

## Results
- Updated `bigclaw-go/internal/planning/planning.go` to use Go-only validation commands and Go evidence links for the v3 candidate backlog.
- Updated `bigclaw-go/internal/planning/planning_test.go` to assert the new Go-native backlog contract.
- Added `bigclaw-go/internal/regression/planning_python_test_replacement_test.go` to block reintroduction of removed Python test references into the planning backlog.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1098/bigclaw-go && go test ./internal/planning ./internal/regression` -> exit `0`
  - `ok  	bigclaw-go/internal/planning	0.941s`
  - `ok  	bigclaw-go/internal/regression	2.392s`
- `rg -n "pytest|tests/test_.*\\.py" bigclaw-go/internal/planning/planning.go bigclaw-go/internal/regression/planning_python_test_replacement_test.go` -> exit `0`
  - matches remain only inside the regression guard that asserts removed Python test paths stay absent and disallowed in backlog validation commands.
