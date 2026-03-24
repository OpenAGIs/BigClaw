## Plan

- Survey the repository to map the parallel evidence bundle index/export flows and performance/benchmark tooling referenced by BIGCLAW-192.
- Optimize the live validation archive index path so canonical timestamped bundle directories stop scanning every retained `summary.json`.
- Add a repo-native benchmark harness for parallel evidence bundle archive export/index generation.
- Update the validation docs with the benchmark entrypoint and record exact validation results.

## Acceptance

- `bigclaw-go/scripts/e2e/export_validation_bundle.py` builds recent archive index entries without full-history summary scans when bundle directories use canonical timestamp run IDs.
- Repository includes a scoped benchmark harness for parallel evidence bundle archive export/index generation.
- Validation docs include the benchmark command.
- `.symphony/workpad.md` documents the plan, acceptance, and validation steps for this issue.
- All targeted tests/benchmarks execute successfully with commands/results captured.

## Validation

- `python3 -m pytest tests/test_parallel_validation_bundle.py tests/test_parallel_validation_bundle_benchmark.py`
  - Result: `3 passed in 0.09s`
- `python3 bigclaw-go/scripts/e2e/benchmark_validation_bundle_export.py --iterations 3 --archive-runs 64 --pretty`
  - Result: exit `0`
  - Timing summary: `min=2.962ms`, `median=3.171ms`, `mean=3.336ms`, `p95=3.875ms`, `max=3.875ms`
