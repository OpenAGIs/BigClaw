# Parallel Validation Matrix (Local/Kubernetes/Ray)

## Scope

This artifact is the repo-native matrix for the current parallel validation
surfaces that are expected to run in BigClaw mainline:

- local executor (`local`)
- Kubernetes executor (`kubernetes`)
- Ray executor (`ray`)

The matrix below is grounded in checked-in evidence under
`bigclaw-go/docs/reports/`.

## Baseline Snapshot

- Baseline run: `20260316T140138Z`
- Source: `bigclaw-go/docs/reports/live-validation-summary.json`
- Summary status: `succeeded`

## Validation Matrix

| Surface | Target surface | Current readiness | Evidence status | Validation commands | Evidence/report paths | Notable caveats |
| --- | --- | --- | --- | --- | --- | --- |
| `local` | SQLite-backed local execution path (`required_executor=local`) | `live_proven` | Canonical smoke report is present and marks task state `succeeded`; bundled report is indexed in live validation summary. | `cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=1 BIGCLAW_E2E_RUN_KUBERNETES=0 BIGCLAW_E2E_RUN_RAY=0 ./scripts/e2e/run_all.sh` | `bigclaw-go/docs/reports/sqlite-smoke-report.json`; `bigclaw-go/docs/reports/live-validation-summary.json`; `bigclaw-go/docs/reports/live-validation-index.md` | Smoke path validates single-task execution and replay-safe event shape; multi-node takeover coverage remains tracked separately. |
| `kubernetes` | Live Kubernetes job execution path (`required_executor=kubernetes`) | `live_proven` | Canonical smoke report is present and marks task state `succeeded`; Kubernetes resources are captured in a separate live resource report. | `cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=0 BIGCLAW_E2E_RUN_KUBERNETES=1 BIGCLAW_E2E_RUN_RAY=0 ./scripts/e2e/run_all.sh`; `cd bigclaw-go && ./scripts/e2e/kubernetes_smoke.sh` | `bigclaw-go/docs/reports/kubernetes-live-smoke-report.json`; `bigclaw-go/docs/reports/kubernetes-live-resources.txt`; `bigclaw-go/docs/reports/live-validation-summary.json` | Requires reachable cluster + valid kube context; smoke result is environment-dependent and should be rerun for new clusters. |
| `ray` | Live Ray job execution path (`required_executor=ray`) | `live_proven` | Canonical smoke report is present and marks task state `succeeded`; Ray job snapshot evidence is checked in. | `cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=0 BIGCLAW_E2E_RUN_KUBERNETES=0 BIGCLAW_E2E_RUN_RAY=1 ./scripts/e2e/run_all.sh`; `cd bigclaw-go && ./scripts/e2e/ray_smoke.sh` | `bigclaw-go/docs/reports/ray-live-smoke-report.json`; `bigclaw-go/docs/reports/ray-live-jobs.json`; `bigclaw-go/docs/reports/live-validation-summary.json` | Requires reachable Ray endpoint/runtime; smoke result validates submission/execution path, not full mixed-workload fairness under sustained load. |

## Companion Evidence

- `bigclaw-go/docs/reports/multi-node-shared-queue-report.json`
- `bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json`
- `bigclaw-go/docs/reports/cross-process-coordination-capability-surface.json`
- `bigclaw-go/docs/reports/cross-process-coordination-boundary-digest.md`
- `bigclaw-go/docs/reports/validation-bundle-continuation-policy-gate.json`

These artifacts remain the companion proofs for shared-queue takeover safety,
cross-process boundary expectations, and validation-bundle continuation policy.

## Canonical follow-up routing

- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining takeover, coordination, continuation, and broker-durability
  hardening lanes behind this validation matrix.
- Start here for runnable lane commands and checked-in evidence, then use the
  follow-up index for `OPE-269` / `BIG-PAR-080`, `OPE-261` / `BIG-PAR-085`,
  `OPE-271` / `BIG-PAR-082`, and `OPE-222`.
