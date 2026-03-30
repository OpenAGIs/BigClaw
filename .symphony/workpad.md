Issue: BIG-GO-1019

Plan
- Inspect `bigclaw-go/scripts/benchmark/**` Python residue and its repo references to identify a self-contained migration slice.
- Replace the benchmark Python CLIs with repo-native Go implementations, preserving current outputs and report shapes so existing docs/evidence stay valid.
- Update direct references in scripts/docs/tests that still point at the removed Python entrypoints.
- Run targeted validation for the migrated benchmark slice, capture exact commands and results, then commit and push the scoped branch changes.

Acceptance
- Changes stay scoped to `bigclaw-go/scripts/**` benchmark residue plus directly coupled references/tests/docs.
- `.py` file count under `bigclaw-go/scripts/benchmark/**` is reduced as much as feasible for this tranche.
- Repo behavior for benchmark matrix generation, local soak invocation, and capacity certification remains available through Go-native entrypoints.
- Final report states the impact on `py files`, `go files`, `pyproject.toml`, and `setup.py`.

Validation
- `find bigclaw-go/scripts -path 'bigclaw-go/scripts/benchmark/*' \( -name '*.py' -o -name '*.go' -o -name '*.sh' \) | sort`
- `go test ./cmd/bigclawctl/... ./scripts/benchmark/...`
- `go run ./cmd/bigclawctl automation benchmark soak-local --help`
- `go run ./scripts/benchmark/run_matrix.go --scenario 50:8 --report-path docs/reports/benchmark-matrix-report.tmp.json`
- `go run ./scripts/benchmark/capacity_certification.go --output bigclaw-go/docs/reports/capacity-certification-matrix.tmp.json --markdown-output bigclaw-go/docs/reports/capacity-certification-report.tmp.md --pretty`
- `git diff --stat && git status --short`
