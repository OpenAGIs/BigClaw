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
- Updated `src/bigclaw/planning.py` so the Python-side v3 candidate backlog now points at Go-native validation commands and Go evidence targets instead of deleted Python tests.
- Extended `bigclaw-go/internal/regression/planning_python_test_replacement_test.go` to scan `src/bigclaw/planning.py` and fail if removed Python test assets reappear there.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1098/bigclaw-go && go test ./internal/planning ./internal/regression` -> exit `0`
  - `ok  	bigclaw-go/internal/planning	(cached)`
  - `ok  	bigclaw-go/internal/regression	0.946s`
- `rg -n "pytest|tests/test_.*\\.py" src/bigclaw/planning.py bigclaw-go/internal/planning/planning.go bigclaw-go/internal/regression/planning_python_test_replacement_test.go` -> exit `0`
  - matches remain only inside the regression guard that enumerates disallowed Python test assets.
- Updated `scripts/dev_bootstrap.sh` so the `BIGCLAW_ENABLE_LEGACY_PYTHON=1` path runs `go test ./internal/bootstrap ./internal/planning ./internal/regression` instead of attempting to execute deleted Python test files.
- Updated `README.md` to replace active `pytest tests` guidance with the Go-native planning/regression replacement commands.
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1098/bigclaw-go && go test ./internal/bootstrap ./internal/planning ./internal/regression` -> exit `0`
  - `ok  	bigclaw-go/internal/bootstrap	3.196s`
  - `ok  	bigclaw-go/internal/planning	(cached)`
  - `ok  	bigclaw-go/internal/regression	(cached)`
- `bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1098/scripts/dev_bootstrap.sh` -> exit `0`
  - `ok  	bigclaw-go/cmd/bigclawctl	3.453s`
  - `BigClaw Go development environment is ready.`
  - `Set BIGCLAW_ENABLE_LEGACY_PYTHON=1 to add the remaining Go-native migration planning coverage after the default Go smoke and bootstrap coverage.`
- `BIGCLAW_ENABLE_LEGACY_PYTHON=1 bash /Users/openagi/code/bigclaw-workspaces/BIG-GO-1098/scripts/dev_bootstrap.sh` -> exit `0`
  - `ok  	bigclaw-go/cmd/bigclawctl	3.857s`
  - `smoke_ok local`
  - `ok  	bigclaw-go/internal/bootstrap	(cached)`
  - `ok  	bigclaw-go/internal/planning	(cached)`
  - `ok  	bigclaw-go/internal/regression	(cached)`
  - `BigClaw Go environment is ready, and the remaining migration planning surface was validated with Go coverage.`
