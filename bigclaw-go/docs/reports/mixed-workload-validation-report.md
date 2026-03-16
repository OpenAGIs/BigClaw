# Mixed Workload Validation Report

## Scope

- Run date: 2026-03-13
- Command: `python3 scripts/e2e/mixed_workload_matrix.py --report-path docs/reports/mixed-workload-matrix-report.json`
- Goal: validate more production-like routing and executor behavior inside one BigClaw Go control-plane instance.
- Artifact contract: keep one stable latest summary plus one timestamped per-run evidence bundle with scenario drilldowns.

## Matrix

- `local-default` -> expected `local`, routed `local`, final state `succeeded`
- `browser-auto` -> expected `kubernetes`, routed `kubernetes`, final state `succeeded`
- `gpu-auto` -> expected `ray`, routed `ray`, final state `succeeded`
- `high-risk-auto` -> expected `kubernetes`, routed `kubernetes`, final state `succeeded`
- `required-ray` -> expected `ray`, routed `ray`, final state `succeeded`

Each scenario emitted `scheduler.routed`, preserved `trace_id`, and reached `task.completed` on the expected executor path. The latest JSON summary now stores normalized expectation labels and links each scenario to its own drilldown artifact instead of embedding all raw events inline.

## Meaning

This matrix gives the epic a same-day mixed-workload proof point rather than isolated single-executor smokes only. It verifies that automatic routing by required tools and risk level, plus explicit executor pinning, all behave as expected against real local/Kubernetes/Ray execution paths.

## Artifact

- `docs/reports/mixed-workload-matrix-report.json`
- `docs/reports/mixed-workload-runs/<run-id>/summary.json`
- `docs/reports/mixed-workload-runs/<run-id>/tasks/<scenario>.json`
- `docs/reports/mixed-workload-runs/<run-id>/README.md`
