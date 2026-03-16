# Benchmark Readiness Report

## Scope

- Run date: 2026-03-13
- Commands:
  - `python3 scripts/benchmark/run_matrix.py --scenario 50:8 --scenario 100:12 --report-path docs/reports/benchmark-matrix-report.json`
  - `python3 scripts/benchmark/soak_local.py --autostart --count 2000 --workers 24 --timeout-seconds 480 --report-path docs/reports/soak-local-2000x24.json`
- Goal: use `OPE-254` to collapse the local benchmark baseline, soak snapshots, and long-duration proof into one reviewer-facing closeout surface.

## Closeout Surface

- Canonical reviewer summary: `docs/reports/benchmark-readiness-report.md`
- Baseline microbenchmark source: `docs/reports/benchmark-report.md`
- Long-duration soak source: `docs/reports/long-duration-soak-report.md`
- Consistency check: `python3 scripts/benchmark/validate_closeout.py`

This closeout stays intentionally local. It consolidates current repo-native benchmark and soak evidence into one auditable pack without claiming production capacity certification.

## Benchmark Snapshot

- `BenchmarkMemoryQueueEnqueueLease-8`: `66075 ns/op`
- `BenchmarkFileQueueEnqueueLease-8`: `31627767 ns/op`
- `BenchmarkSQLiteQueueEnqueueLease-8`: `18057898 ns/op`
- `BenchmarkSchedulerDecide-8`: `73.98 ns/op`

These results keep queue and scheduler microbenchmarks in the same local-dev performance band as the earlier baseline captured in `docs/reports/benchmark-report.md` while adding a repeatable matrix runner for future review passes.

## Soak Matrix

- `50 tasks x 8 workers`: `8.232s`, `6.074 tasks/s`, `50 succeeded`, `0 failed`
- `100 tasks x 12 workers`: `10.294s`, `9.714 tasks/s`, `100 succeeded`, `0 failed`
- `1000 tasks x 24 workers`: `104.091s`, `9.607 tasks/s`, `1000 succeeded`, `0 failed`
- `2000 tasks x 24 workers`: `219.167s`, `9.125 tasks/s`, `2000 succeeded`, `0 failed`

Every sampled task reached `task.completed`, preserved `trace_id`, and emitted `scheduler.routed` during the soak runs.

## Artifacts

- `docs/reports/benchmark-matrix-report.json`
- `docs/reports/soak-local-50x8.json`
- `docs/reports/soak-local-100x12.json`
- `docs/reports/soak-local-1000x24.json`
- `docs/reports/soak-local-2000x24.json`
- `docs/reports/long-duration-soak-report.md`
- `docs/reports/benchmark-report.md`
- `scripts/benchmark/run_matrix.py`
- `scripts/benchmark/validate_closeout.py`

## Readiness

`OPE-254` now has a reproducible local matrix runner, refreshed queue/scheduler benchmark output, four soak proof points with zero failures, and a repo-native validator that checks the markdown closeout against the underlying JSON and benchmark stdout artifacts. That is enough to close the current benchmark scope in Linear while leaving production-grade capacity certification as explicit follow-up hardening work.
