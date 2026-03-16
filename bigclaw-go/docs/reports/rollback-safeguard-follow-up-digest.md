# Rollback Safeguard Follow-up Digest

## Scope

This digest consolidates the repo-native rollback guardrail and tenant-scoped trigger surface for `OPE-267` / `BIG-PAR-093`.

## Current Repo-Backed Evidence

- `docs/reports/migration-readiness-report.md` records rollback as a current migration gap rather than a completed runtime capability.
- `docs/migration.md` captures the current rollback plan and operator steps.
- `docs/reports/migration-plan-review-notes.md` records that rollback is intentionally simple and operator-driven in the current phase.
- `docs/reports/review-readiness.md` explains which migration claims are safe to treat as closure-ready.
- `docs/reports/issue-coverage.md` maps the current migration implementation and its follow-up caveats back to the rewrite issues.

## Tenant-Scoped Trigger Surface

| Trigger surface | Repo-native evidence | Required tenant-scoped operator response | Automation boundary |
| --- | --- | --- | --- |
| Shadow parity drift for the tenant or task slice under rollout | `docs/reports/shadow-compare-report.json`, `docs/reports/shadow-matrix-report.json`, and the migration readiness summary | Pause the tenant rollout segment, stop assigning new Go-owned work for the affected slice, and redirect eligible submissions back to the incumbent plane while the diff is reviewed | Detection is evidence-backed, but the stop/redirect action is still manual |
| Live validation or smoke evidence shows tenant-impacting failures during cutover rehearsal | `docs/reports/live-validation-summary.json` plus the latest run bundle under `docs/reports/live-validation-runs/*` | Hold further tenant expansion, preserve the failed run artifacts, and revert the rollout slice to the incumbent plane before widening exposure | No automatic cutover gate consumes these artifacts yet |
| Queue lease, replay, or event-sequence anomalies make the tenant outcome unauditable | `docs/reports/queue-reliability-report.md`, `docs/reports/lease-recovery-report.md`, and `docs/reports/event-bus-reliability-report.md` | Treat the tenant as rollback-eligible, stop new leases in the Go control plane, and review replay/audit evidence before resuming any cutover step | The repo documents the threshold class, but no runtime policy engine enforces it |
| Audit trail or migration evidence is missing for a tenant-scoped rollout checkpoint | `docs/migration.md`, `docs/reports/migration-plan-review-notes.md`, and this digest | Do not advance the tenant cutover checkpoint; restore traffic to the incumbent plane if the missing evidence blocks safe review | Evidence completeness is a reviewer requirement, not a machine-enforced admission check |

## Reviewer Digest

- Rollback is currently a manual, evidence-backed operator action, not an automated tenant-scoped safeguard.
- The documented rollback path remains configuration- and routing-driven: disable Go dispatch, stop new leases, and redirect eligible traffic back to the incumbent plane.
- Current cutover protection relies on preserving audit trails, shadow evidence, and live validation artifacts for review instead of on an automatic rollback trigger.
- This is sufficient for phased rollout planning, but it is not yet a policy-enforced rollback control plane.

## Manual Rollback Guardrails

- Tenant-scoped rollback starts by pausing rollout expansion for the affected tenant, task type, or environment before changing any broader routing default.
- Operators must preserve the relevant shadow, live-validation, and audit artifacts before redirecting traffic so the failed checkpoint remains reviewable after rollback.
- Redirecting traffic back to the incumbent plane is the rollback action; deleting evidence, mutating historical reports, or silently retrying the same cutover step is not.
- Future automation may consume the same trigger surface, but this digest only describes the current reviewable guardrails and manual steps.

## Current Blockers

- No tenant-scoped automated rollback trigger exists yet.
- No cutover-protection gate automatically halts risky rollout segments.
- No rollback scorecard ties live migration drift directly to traffic-disable automation.
- No policy engine yet coordinates rollback thresholds across local, Kubernetes, and Ray rollout surfaces.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/migration-readiness-report.md`, `docs/migration.md`, and `docs/reports/migration-plan-review-notes.md`.
- Repeat the `rollback remains operator-driven`, `manual, evidence-backed operator action`, and `no tenant-scoped automated rollback trigger` caveats anywhere migration safety is summarized.
- When rollback safeguards become policy-driven, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
