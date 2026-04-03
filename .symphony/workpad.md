# BIG-GO-1145

## Plan

- Confirm the current repository Python inventory and identify whether the listed candidate files still exist.
- Align issue-scoped regression and migration documentation with the current Go-only state for the affected script surfaces.
- Run targeted validation for the replacement Go commands and the regression coverage that guards the removed Python paths.
- Commit the scoped changes and push the branch to the remote.

## Acceptance

- The lane-covered Python entrypoints remain absent from the repository.
- Go replacement and compatibility paths for the covered automation surfaces are still present and validated.
- Repository assertions and migration documentation no longer imply that the removed Python files still exist or still need migration.
- Validation records the exact commands and results for the issue-scoped checks.

## Validation

- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help`

## Validation Results

- `find . -name '*.py' | wc -l` -> `0`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/regression` -> `ok  	bigclaw-go/cmd/bigclawctl	(cached)`; `ok  	bigclaw-go/internal/regression	0.515s`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation --help` -> exit `0`; printed `usage: bigclawctl automation <e2e|benchmark|migration> ...`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke --help` -> exit `0`; printed `usage: bigclawctl automation e2e run-task-smoke [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e export-validation-bundle --help` -> exit `0`; printed `usage: bigclawctl automation e2e export-validation-bundle [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help` -> exit `0`; printed `usage: bigclawctl automation benchmark run-matrix [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help` -> exit `0`; printed `usage: bigclawctl automation benchmark capacity-certification [flags]`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation migration shadow-compare --help` -> exit `0`; printed `usage: bigclawctl automation migration shadow-compare [flags]`

## Notes

- The repository already started from a zero-`*.py` baseline in this workspace, so this issue could only harden deletion enforcement and stale-migration documentation for the lane; it could not reduce the Python-file count below zero.
