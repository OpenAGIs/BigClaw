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
- migration readiness comparison

## Local matrix helper

```bash
cd bigclaw-go
python3 scripts/benchmark/run_matrix.py \
  --scenario 50:8 \
  --scenario 100:12 \
  --report-path docs/reports/benchmark-matrix-report.json
```


## Long-duration soak helper

```bash
cd bigclaw-go
python3 scripts/benchmark/soak_local.py \
  --autostart \
  --count 2000 \
  --workers 24 \
  --timeout-seconds 480 \
  --report-path docs/reports/soak-local-2000x24.json
```
