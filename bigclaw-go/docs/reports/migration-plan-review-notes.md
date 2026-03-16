# Migration Plan Review Notes

## Scope

This note captures the review outcome for the Go rewrite boundary and migration plan defined by:

- `docs/adr/0001-go-rewrite-control-plane.md`
- `docs/migration.md`
- `docs/migration-shadow.md`
- `scripts/migration/shadow_compare.py`
- `scripts/migration/check_review_pack.py`

## Boundary Decisions

- The Go control plane owns queueing, scheduling, worker execution lifecycle, executor adapters, event fan-out, and audit/observability entrypoints.
- Existing task producers and higher-level workflow integrations remain compatibility clients of the Go control plane rather than being rewritten in the same phase.
- Kubernetes and Ray integrations are first-class executor adapters behind a shared task protocol.

## Migration Path

1. Stand up the Go control plane alongside the incumbent implementation.
2. Run shadow comparison and timeline diffing using `scripts/migration/shadow_compare.py`.
3. Confirm matrix coverage through `docs/reports/shadow-matrix-report.json`.
4. Validate queue, scheduler, worker, Kubernetes, and Ray paths through the latest indexed bundle in `docs/reports/live-validation-index.md`.
5. Run `python3 scripts/migration/check_review_pack.py` so the review-pack references and artifact paths stay in sync.
6. Roll traffic gradually by tenant, task type, or environment once the repo-native evidence pack remains green.
7. Use rollback by redirecting submission back to the incumbent control plane and preserving the stored review-pack artifacts for postmortem review.

## Rollback Notes

- The Go validation path is isolated enough to be disabled without deleting existing queue or audit reports.
- Live validation reports persist enough metadata (`base_url`, `state_dir`, `service_log`) to reconstruct a failed rollout attempt.
- Shadow compare and shadow matrix output provide low-risk read-only verification steps before cutover.

## Review-Pack Evidence

- `docs/reports/migration-readiness-report.md` is the stable migration closeout entrypoint for reviewers.
- `docs/reports/shadow-compare-report.json` records a matched sample run on `2026-03-13`.
- `docs/reports/shadow-matrix-report.json` records `3/3` matched sample fixtures on `2026-03-13`.
- `docs/reports/live-validation-index.md` records a succeeded latest bundle for run `20260314T164647Z` generated on `2026-03-14T16:46:57.671520+00:00`.

## Review Outcome

- ADR boundary is internally consistent with the current code layout and runtime wiring.
- Migration plan supports shadow validation and live smoke review before cutover.
- Rollback path is documented and operationally simple.
- Closeout claims should remain limited to the stored repo-native evidence in the migration review pack.
