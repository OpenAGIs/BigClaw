## Plan

1. Confirm the current benchmark helper state under `bigclaw-go/scripts/benchmark/` and enumerate every remaining README / workflow / hook / CI reference to the removed Python benchmark entrypoints.
2. Update benchmark-facing docs and wrappers so the repository points only at the Go-owned `bigclawctl automation benchmark ...` commands and the retained `run_suite.sh` wrapper.
3. Run targeted validation for the benchmark Go CLI commands and the benchmark wrapper script, plus before/after `.py` file counts for the benchmark directory and repository.
4. Commit the scoped `BIG-GO-1051` changes and push the branch to the remote.

## Acceptance

- No tracked Python files remain under `bigclaw-go/scripts/benchmark/`.
- Repository entrypoints no longer tell operators to invoke `bigclaw-go/scripts/benchmark/*.py`.
- Benchmark documentation points at `go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification` or `scripts/benchmark/run_suite.sh`.
- Any benchmark-related workflow / hook / CI references in the touched scope use Go entrypoints only.
- Validation records the exact commands and outcomes for benchmark help commands, the benchmark suite wrapper, and `.py` file counts.
- Changes are committed and pushed on the current branch.

## Validation

- `find bigclaw-go/scripts/benchmark -name '*.py' | wc -l`
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && ./scripts/benchmark/run_suite.sh`
- `git status --short`
- `git log -1 --stat`
