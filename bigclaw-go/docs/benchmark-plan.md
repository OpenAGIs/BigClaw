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

## Local matrix helper

```bash
cd bigclaw-go
go test -bench . ./internal/queue ./internal/scheduler
```

Use the checked-in matrix at `docs/reports/benchmark-matrix-report.json` for the
repo-native benchmark slice documented by this plan.


## Long-duration soak helper

```bash
cat bigclaw-go/docs/reports/soak-local-2000x24.json
```

The active evidence path is the checked-in soak corpus under `docs/reports/`.

## Capacity certification helper

```bash
cat bigclaw-go/docs/reports/capacity-certification-matrix.json
```

This helper converts the checked-in benchmark, soak, and mixed-workload artifacts into
an explicit certification matrix with pass/fail thresholds, saturation notes, and
recommended operating envelopes. It is still repo-native evidence rather than a live
production attestation.
