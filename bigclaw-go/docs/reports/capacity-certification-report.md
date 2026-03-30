# Capacity Certification Report

## Scope

- Generated at: `2026-03-13T09:44:42.458392Z`
- Ticket: `BIG-PAR-098`
- Goal: convert checked-in benchmark, soak, and mixed-workload evidence into a repo-native certification matrix with explicit thresholds and operating envelopes.
- Boundary: this is a single-instance repo-native certification slice, not a live multi-tenant production attestation.

## Certification Summary

- Overall status: `pass`
- Passed lanes: `9/9`
- Recommended local sustained envelope: `<=1000 tasks with 24 submit workers`
- Local ceiling envelope: `<=2000 tasks with 24 submit workers`
- Saturation signal: `throughput remains in the same single-instance local band at the 2000-task ceiling`

## Admission Policy Summary

- Policy mode: `advisory-only reviewer guidance`
- Runtime enforcement: `none`
- Default reviewer envelope: `<=1000 tasks with 24 submit workers`
- Ceiling reviewer envelope: `<=2000 tasks with 24 submit workers`
- Scheduler note: recommended envelopes guide reviewer admission decisions and are not scheduler-enforced runtime limits.

## Microbenchmark Thresholds

- `BenchmarkFileQueueEnqueueLease-8`: `31627767.00 ns/op` vs limit `4e+07` -> `pass`
- `BenchmarkMemoryQueueEnqueueLease-8`: `66075.00 ns/op` vs limit `100000` -> `pass`
- `BenchmarkSQLiteQueueEnqueueLease-8`: `18057898.00 ns/op` vs limit `2.5e+07` -> `pass`
- `BenchmarkSchedulerDecide-8`: `73.98 ns/op` vs limit `1000` -> `pass`

## Soak Matrix

- `50x8`: `6.074 tasks/s`, `0 failed`, envelope `bootstrap-burst` -> `pass`
- `100x12`: `9.714 tasks/s`, `0 failed`, envelope `bootstrap-burst` -> `pass`
- `1000x24`: `9.607 tasks/s`, `0 failed`, envelope `recommended-local-sustained` -> `pass`
- `2000x24`: `9.125 tasks/s`, `0 failed`, envelope `recommended-local-ceiling` -> `pass`

## Workload Mix

- `mixed-workload-routing`: `all sampled mixed-workload routes landed on the expected executor path` -> `pass`

## Recommended Operating Envelopes

- `recommended-local-sustained`: Use up to 1000 queued tasks with 24 submit workers when a stable single-instance local review lane is required. Evidence: `1000x24`.
- `recommended-local-ceiling`: Treat 2000 queued tasks with 24 submit workers as the checked-in local ceiling, not the default operating point. Evidence: `2000x24`.
- `mixed-workload-routing`: Use the mixed-workload matrix for executor routing correctness, but do not infer sustained multi-executor throughput from it. Evidence: `mixed-workload-routing`.

## Saturation Notes

- Throughput plateaus around 9-10 tasks/s across the checked-in 100x12, 1000x24, and 2000x24 local lanes.
- The 2000x24 lane remains within the same throughput band as 1000x24, so the checked-in local ceiling is evidence-backed but not substantially headroom-rich.
- Mixed-workload evidence verifies executor-routing correctness across local, Kubernetes, and Ray, but it is a functional routing proof rather than a concurrency ceiling.

## Limits

- Evidence is repo-native and single-instance; it does not certify multi-node or multi-tenant production saturation behavior.
- The matrix uses checked-in local runs from 2026-03-13 and should be refreshed when queue, scheduler, or executor behavior changes materially.
- Recommended envelopes are conservative reviewer guidance derived from current evidence, not an automated runtime admission policy.
