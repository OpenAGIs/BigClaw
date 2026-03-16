# Benchmark Readiness Report

## Scope

- Run date: 2026-03-16
- Commands:
  - `python3 scripts/benchmark/run_matrix.py --scenario 50:8 --scenario 100:12 --report-path docs/reports/benchmark-matrix-report.json`
  - `python3 scripts/benchmark/run_matrix.py --validate-report docs/reports/benchmark-matrix-report.json`
- Goal: normalize benchmark matrix outputs around one canonical artifact surface so readiness and rollout review can link stable benchmark and soak evidence.

## Benchmark Snapshot

- `BenchmarkMemoryQueueEnqueueLease-8`: `169960 ns/op`
- `BenchmarkFileQueueEnqueueLease-8`: `404190139 ns/op`
- `BenchmarkSQLiteQueueEnqueueLease-8`: `74385807 ns/op`
- `BenchmarkSchedulerDecide-8`: `1048 ns/op`

The canonical JSON artifact now exposes benchmark cases and soak reports under one stable `artifacts` surface while preserving the raw benchmark stdout and soak payloads for detailed follow-up review.

## Soak Matrix

- `50 tasks x 8 workers`: `11.143s`, `4.487 tasks/s`, `50 succeeded`, `0 failed`
- `100 tasks x 12 workers`: `10.221s`, `9.784 tasks/s`, `100 succeeded`, `0 failed`
- `1000 tasks x 24 workers`: `104.091s`, `9.607 tasks/s`, `1000 succeeded`, `0 failed`
- `2000 tasks x 24 workers`: `219.167s`, `9.125 tasks/s`, `2000 succeeded`, `0 failed`

Every matrix scenario exported a canonical soak artifact entry with stable `scenario_id`, report path, throughput, and pass/fail status. Sampled tasks still reached `task.completed`, preserved `trace_id`, and emitted `scheduler.routed` during the refreshed matrix runs.

## Canonical Artifacts

- `docs/reports/benchmark-matrix-report.json`
- `docs/reports/benchmark-readiness-report.md`
- `docs/reports/benchmark-report.md`
- `docs/reports/soak-local-50x8.json`
- `docs/reports/soak-local-100x12.json`
- `docs/reports/soak-local-1000x24.json`
- `docs/reports/soak-local-2000x24.json`
- `scripts/benchmark/run_matrix.py`

## Readiness

`OPE-186` now has one repo-native benchmark matrix artifact that indexes benchmark cases, scenario-level soak reports, and the paired readiness markdown without rebuilding the bundle manually. The refreshed `50x8` and `100x12` matrix scenarios both validated cleanly on 2026-03-16, and the existing `1000x24` plus `2000x24` long-duration soak artifacts remain part of the same canonical benchmark readiness surface.
