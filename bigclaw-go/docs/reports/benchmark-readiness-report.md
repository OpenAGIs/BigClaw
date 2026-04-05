# Benchmark Readiness Report

## Scope

- Run date: 2026-03-13
- Commands:
  - `python3 scripts/benchmark/run_matrix.py --scenario 50:8 --scenario 100:12 --report-path docs/reports/benchmark-matrix-report.json`
  - `python3 scripts/benchmark/soak_local.py --autostart --count 2000 --workers 24 --timeout-seconds 480 --report-path docs/reports/soak-local-2000x24.json`
- Goal: refresh `OPE-186` with a repeatable local benchmark matrix plus concurrent and longer-duration soak evidence.
- Evidence class: bootstrap proof for local benchmark health, not the final certification artifact.

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
- `docs/reports/capacity-certification-matrix.json`
- `docs/reports/capacity-certification-report.md`
- `scripts/benchmark/run_matrix.py`
- `scripts/benchmark/capacity_certification/main.go`

## Bootstrap Meaning

`OPE-186` now has a reproducible local matrix runner, refreshed queue/scheduler benchmark output, and four soak proof points with zero failures, including a `1k+` burst and a longer `2000x24` run. This is enough to close the current benchmark scope in Linear, while production-grade capacity certification can remain follow-up hardening work.

## Certification Follow-through

`BIG-PAR-098` turns this bootstrap package into a checked-in capacity certification matrix. The certification layer makes the current evidence boundary explicit:

- `50x8` and `100x12` remain bootstrap burst lanes.
- `1000x24` is the recommended single-instance local sustained envelope.
- `2000x24` is the current checked-in local ceiling, with throughput staying in the same `9-10 tasks/s` band and no failures.
- Mixed local / Kubernetes / Ray executor routing is certified for correctness via `docs/reports/mixed-workload-matrix-report.json`, but not for sustained multi-executor saturation.

## Canonical follow-up routing

- `docs/reports/capacity-certification-report.md` and
  `docs/reports/capacity-certification-matrix.json` remain the checked-in
  capacity-certification entrypoints for the benchmark envelope summarized here.
- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining coordination, takeover, continuation, and broker-durability
  hardening lanes that still sit outside this closed local benchmark baseline.
- When this benchmark proof is reviewed alongside executor or migration
  readiness, use the follow-up index for `OPE-269` / `BIG-PAR-080`,
  `OPE-261` / `BIG-PAR-085`, `OPE-271` / `BIG-PAR-082`, and `OPE-222`.
