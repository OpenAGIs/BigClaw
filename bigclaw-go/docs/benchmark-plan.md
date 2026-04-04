# BigClaw Go Benchmark Plan

## Scenarios

- 100 tasks burst
- 1k tasks burst
- mixed retry storm
- lease expiry recovery
- high-risk Kubernetes routing
- Ray fan-out style routing

## Metrics

- enqueue latency
- lease latency
- scheduler decision latency
- retry recovery latency
- queue persistence overhead
- execution success ratio

## Outputs

- queue benchmark report
- scheduler benchmark report
- soak test report
- capacity certification matrix
- migration readiness comparison

## Local matrix evidence

Use the checked-in benchmark matrix in `docs/reports/benchmark-matrix-report.json`
for the canonical `50:8` and `100:12` local burst evidence. The retained
`scripts/benchmark/run_suite.sh` wrapper still refreshes the Go microbenchmark
report in `docs/reports/benchmark-report.md`.


## Long-duration soak evidence

Use the checked-in soak evidence in `docs/reports/soak-local-1000x24.json` and
`docs/reports/soak-local-2000x24.json` for the current sustained local envelope.

## Capacity certification evidence

Use `docs/reports/capacity-certification-matrix.json` and
`docs/reports/capacity-certification-report.md` as the canonical checked-in
certification surface for the current benchmark, soak, and mixed-workload
evidence bundle.

This helper converts the checked-in benchmark, soak, and mixed-workload artifacts into
an explicit certification matrix with pass/fail thresholds, saturation notes, and
recommended operating envelopes. It is still repo-native evidence rather than a live
production attestation.
