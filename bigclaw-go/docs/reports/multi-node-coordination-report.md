# Multi-Node Coordination Report

## Scope

- Run date: 2026-03-13
- Command: `go run ./cmd/bigclawctl automation e2e multi-node-shared-queue --count 200 --submit-workers 8 --timeout-seconds 180 --report-path docs/reports/multi-node-shared-queue-report.json`
- Goal: produce a concrete two-node shared-queue proof for the current Go control plane.

## Result

- Total tasks: `200`
- Submitted by `node-a`: `100`
- Submitted by `node-b`: `100`
- Completed by `node-a`: `73`
- Completed by `node-b`: `127`
- Cross-node completions: `99`
- Duplicate `task.started`: `0`
- Duplicate `task.completed`: `0`
- Missing terminal completions: `0`

## Meaning

This run proves that two independent `bigclawd` processes can share the same SQLite-backed queue and coordinate task consumption without duplicate terminal execution in the current local topology. The repo now also exposes an explicit leader-election scaffold through `/coordination/leader`, `/debug/status` (`coordination_leader_election`), and `/v2/control-center` (`coordination_leader_election`) so coordinator ownership is modeled directly instead of being inferred only from queue behavior. The hardening remains SQLite-backed and local/shared-store scoped rather than a broker-backed or quorum-backed leader-election system.

The dedicated leader-election capability matrix in `docs/reports/leader-election-capability-surface.json` now makes the backend posture explicit: the shared SQLite subscriber-lease backend is `live_proven`, shared-store takeover hardening is `harness_proven`, and broker-backed or quorum-backed ownership remains `contract_only`.

In the runtime capability matrix, this shared-queue result is the current `live_proven` shared-queue proof. Subscriber takeover, stale-writer fencing, and replay coordination remain `harness_proven` or `contract_only` until the same semantics are emitted by a live multi-node run.

## Artifact

- `docs/reports/multi-node-shared-queue-report.json`
- `docs/reports/shared-queue-companion-summary.json`
- `docs/reports/live-validation-index.md`
- `docs/reports/cross-process-coordination-capability-surface.json`
- `docs/reports/leader-election-capability-surface.json`

## Parallel Follow-up Index

- `docs/reports/parallel-follow-up-index.md` is the canonical index for the
  remaining coordination, takeover, and validation-continuation caveats behind
  the current multi-node proof set.
- Use `docs/reports/parallel-validation-matrix.md` for the checked-in
  local/Kubernetes/Ray validation entrypoint before drilling into the follow-up
  caveats.
- The two coordination follow-up digests most directly tied to this report are
  `docs/reports/cross-process-coordination-boundary-digest.md` and
  `OPE-271` / `BIG-PAR-082` in
  `docs/reports/validation-bundle-continuation-digest.md`.
