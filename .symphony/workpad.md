## Plan

1. Confirm the benchmark helper state under `bigclaw-go/scripts/benchmark/` and enumerate any remaining README / workflow / hook / CI references to removed benchmark Python entrypoints.
2. Keep benchmark-facing docs and wrappers pointed only at the Go-owned `bigclawctl automation benchmark ...` commands and the retained `run_suite.sh` wrapper.
3. Run targeted validation for the benchmark Go CLI commands, the benchmark suite wrapper, and benchmark-related `.py` file counts.
4. Record validation, closeout, and machine-readable status artifacts for `BIG-GO-1051`, then commit and push the lane.

## Acceptance

- No tracked Python files remain under `bigclaw-go/scripts/benchmark/`.
- Repository entrypoints do not tell operators to invoke `bigclaw-go/scripts/benchmark/*.py`.
- Benchmark documentation points at `go run ./cmd/bigclawctl automation benchmark soak-local|run-matrix|capacity-certification` or `scripts/benchmark/run_suite.sh`.
- Any benchmark-related workflow / hook / CI references in the touched scope use Go entrypoints only.
- Validation records the exact commands and outcomes for benchmark help commands, the benchmark suite wrapper, and `.py` file counts.
- Closeout artifacts for `BIG-GO-1051` exist in `reports/`.
- Changes are committed and pushed on `main`.

## Validation

- `find bigclaw-go/scripts/benchmark -name '*.py' | wc -l`
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./cmd/bigclawctl/...`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark soak-local --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark run-matrix --help`
- `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark capacity-certification --help`
- `cd bigclaw-go && ./scripts/benchmark/run_suite.sh`
- `rg -n "bigclaw-go/scripts/benchmark/(soak_local|run_matrix|capacity_certification)\.py|scripts/benchmark/.*\.py|soak_local\.py|run_matrix\.py|capacity_certification\.py" .`
- `python3 -m json.tool reports/BIG-GO-1051-status.json >/dev/null`
- `git status --short --branch`
- `git rev-parse HEAD`
- `git rev-parse origin/main`
