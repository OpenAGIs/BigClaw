# Cross-Process Coordination Boundary Digest

## Scope

This digest consolidates the remaining cross-process coordination caveats for `OPE-261` / `BIG-PAR-085` and the contract-surface follow-up in `OPE-257` / `BIG-PAR-095`.

## Current Repo-Backed Evidence

- `docs/reports/event-bus-reliability-report.md` captures the current event-bus durability shape and remaining coordination gaps.
- `docs/reports/multi-node-coordination-report.md` provides the concrete two-node shared-queue proof currently available in-repo.
- `docs/reports/cross-process-coordination-capability-surface.json` adds a machine-readable runtime capability matrix tying together `live_proven`, `harness_proven`, `contract_only`, and supporting-surface readiness.
- `docs/reports/broker-event-log-adapter-contract.md` records the provider-neutral contract surface for future `PartitionRoute` and `SubscriberOwnershipContract` support.
- `docs/reports/review-readiness.md` records which distributed coordination claims are already safe to treat as closure-ready.
- `docs/reports/issue-coverage.md` records the current event-bus and migration evidence plus the remaining follow-up digests.
- `docs/openclaw-parallel-gap-analysis.md` captures the remaining distributed mainline gaps after the current BigClaw evidence set.

## Reviewer Digest

- The repo now has a machine-readable runtime capability matrix that distinguishes `live_proven`, `harness_proven`, and `contract_only` coordination semantics, plus the supporting metadata surface that reports those boundaries.
- The repo has a concrete shared-queue coordination proof, but broker-backed ownership and partitioned routing remain contract-only targets described through `PartitionRoute` and `SubscriberOwnershipContract`.
- Current coordination evidence is still bounded by local SQLite-backed sharing plus deterministic local takeover harnesses, not a durable broker-backed cross-process subscriber coordination contract.
- There is no shipped partitioned topic routing model, no shipped broker-backed subscriber ownership model, and no provider-neutral live proof for cross-process replay coordination.
- Cross-process coordination is therefore bounded by the current local proof and roadmap documentation, not by a completed runtime contract.
- The current ceiling still includes `no partitioned topic model` and `no broker-backed cross-process subscriber coordination` even after adding the target contract surface.

## Current Blockers

- No runtime partitioned topic model exists yet beyond the contract-only `PartitionRoute` target.
- No runtime broker-backed cross-process subscriber coordination exists yet beyond the contract-only `SubscriberOwnershipContract` target.
- No durable backend currently carries subscriber ownership across independent processes or nodes beyond the shared local SQLite proof.
- No executable evidence bundle yet proves the same coordination guarantees under a replicated transport.

## Lightweight Consistency Check

- Keep this digest aligned with `docs/reports/event-bus-reliability-report.md`, `docs/reports/multi-node-coordination-report.md`, `docs/reports/cross-process-coordination-capability-surface.json`, and `docs/e2e-validation.md`.
- Repeat the contract-only `PartitionRoute` and `SubscriberOwnershipContract` wording anywhere distributed coordination is summarized.
- When cross-process coordination becomes runtime-complete, update this digest, `docs/reports/review-readiness.md`, and `docs/reports/issue-coverage.md` together.
