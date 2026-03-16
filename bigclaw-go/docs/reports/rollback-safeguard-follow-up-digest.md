# Rollback Safeguard Follow-up Digest

## Scope

This digest consolidates the remaining rollback-safeguard caveats for `OPE-267` / `BIG-PAR-078`.

## Current Repo-Backed Evidence

- `docs/reports/migration-readiness-report.md` records rollback as a current migration gap rather than a completed runtime capability.
- `docs/migration.md` captures the current rollback plan and operator steps.
- `docs/reports/migration-plan-review-notes.md` records that rollback is intentionally simple and operator-driven in the current phase.
- `docs/reports/review-readiness.md` explains which migration claims are safe to treat as closure-ready.
- `docs/reports/issue-coverage.md` maps the current migration implementation and its follow-up caveats back to the rewrite issues.

## Reviewer Digest

- Rollback is currently operational guidance, not an automated tenant-scoped safeguard.
- The documented rollback path is configuration- and routing-driven: disable Go dispatch, stop new leases, and redirect eligible traffic back to the incumbent plane.
- Current cutover protection relies on preserving audit trails and shadow evidence for manual review rather than on an automatic rollback trigger.
- This is sufficient for phased rollout planning, but it is not yet a policy-enforced rollback control plane.

## Current Blockers

- No tenant-scoped automated rollback trigger exists yet.
- No cutover-protection gate automatically halts risky rollout segments.
- No rollback scorecard ties live migration drift directly to traffic-disable automation.
- No policy engine yet coordinates rollback thresholds across local, Kubernetes, and Ray rollout surfaces.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/migration-readiness-report.md`, `docs/migration.md`, and `docs/reports/migration-plan-review-notes.md`.
- Repeat the `rollback remains operator-driven` and `no tenant-scoped automated rollback trigger` caveats anywhere migration safety is summarized.
- When rollback safeguards become policy-driven, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
