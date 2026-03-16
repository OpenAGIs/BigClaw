# Migration Plan Review Notes

## Scope

This note captures the review outcome for the Go rewrite boundary and migration plan defined by:

- `docs/adr/0001-go-rewrite-control-plane.md`
- `docs/migration.md`
- `docs/migration-shadow.md`
- `scripts/migration/shadow_compare.py`

## Boundary Decisions

- The Go control plane owns queueing, scheduling, worker execution lifecycle, executor adapters, event fan-out, and audit/observability entrypoints.
- Existing task producers and higher-level workflow integrations remain compatibility clients of the Go control plane rather than being rewritten in the same phase.
- Kubernetes and Ray integrations are first-class executor adapters behind a shared task protocol.

## Migration Path

1. Stand up the Go control plane alongside the incumbent implementation.
2. Run shadow comparison and timeline diffing using `scripts/migration/shadow_compare.py`.
3. Validate queue, scheduler, worker, Kubernetes, and Ray paths through local and live smoke reports.
4. Roll traffic gradually by tenant, task type, or environment.
5. Use rollback by redirecting submission back to the incumbent control plane and preserving Go reports for postmortem review.

## Rollback Notes

- The Go validation path is isolated enough to be disabled without deleting existing queue or audit reports.
- Live validation reports persist enough metadata (`base_url`, `state_dir`, `service_log`) to reconstruct a failed rollout attempt.
- Shadow compare output provides a low-risk read-only verification step before cutover.

## Review Outcome

- ADR boundary is internally consistent with the current code layout and runtime wiring.
- Migration plan supports shadow validation before cutover.
- Rollback path is documented and operationally simple.
- This issue is considered ready for closure from a design-governance perspective.

## Parallel follow-up digests

- `OPE-266` / `BIG-PAR-092` tracks the repo-native live shadow mirror parity drift scorecard and the remaining live shadow traffic comparison caveats in `docs/reports/live-shadow-comparison-follow-up-digest.md`.
- `OPE-267` / `BIG-PAR-078` tracks the remaining rollback safeguard caveats in `docs/reports/rollback-safeguard-follow-up-digest.md`.
