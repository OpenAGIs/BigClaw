# BigClaw Go Mainline Merge Plan

This document is the repo-native merge and compatibility landing plan for
`BIG-GO-910`.

As of March 27, 2026, the historical Go-mainline cutover branch
`symphony/BIG-GOM-302` has already landed on `main` through PR `#138`, which
merged on March 21, 2026 with merge commit
`94e1f455e48a8115249cfa82b047528c010dd495`.

`BIG-GO-910` does not reopen that merged work. It consolidates the executable
path for replaying, verifying, and safely extending the cutover using the
existing repo-native evidence.

## Scope mapping

Treat `BIG-GOM-301` through `BIG-GOM-309` as the first nine implementation and
validation subtasks behind the cutover. Treat `BIG-GOM-310` as the handoff
closeout slice that records final merge readiness and post-merge documentation.

| Subtask | Scope | Canonical evidence |
| --- | --- | --- |
| `BIG-GOM-301` | domain model and intake contract migration | `docs/go-domain-intake-parity-matrix.md`, `bigclaw-go/internal/domain`, `bigclaw-go/internal/intake`, `bigclaw-go/internal/workflow` |
| `BIG-GOM-302` | risk, policy, approval, and audit semantics | `bigclaw-go/internal/governance`, `bigclaw-go/internal/contract`, `bigclaw-go/internal/observability` |
| `BIG-GOM-303` | workflow orchestration and scheduler loop | `bigclaw-go/internal/workflow`, `bigclaw-go/internal/scheduler`, `bigclaw-go/internal/worker`, `bigclaw-go/internal/queue` |
| `BIG-GOM-304` | observability, reporting, and weekly operations | `bigclaw-go/internal/reporting`, `bigclaw-go/internal/regression`, `/v2/reports/*` |
| `BIG-GOM-305` | control center, triage, and operations views | `bigclaw-go/internal/api`, `bigclaw-go/internal/product`, `bigclaw-go/internal/triage` |
| `BIG-GOM-306` | repo collaboration and lineage surfaces | `bigclaw-go/internal/api`, `src/bigclaw/repo_*`, repo-lineage evidence recorded in `local-issues.json` |
| `BIG-GOM-307` | workflow/bootstrap/GitHub-sync toolchain migration | `bigclaw-go/internal/bootstrap`, `scripts/ops/bigclawctl`, `src/bigclaw/workspace_bootstrap.py`, `src/bigclaw/github_sync.py` |
| `BIG-GOM-308` | Python deprecation and Go-only mainline switch | `src/bigclaw/legacy_shim.py`, `docs/go-mainline-cutover-issue-pack.md`, `docs/go-mainline-cutover-handoff.md` |
| `BIG-GOM-309` | branch-wide final validation before merge | full-suite `go test ./...`, legacy shim compile checks, Git sync proof, PR `#138` alignment |

## Historical branch and PR disposition

- Canonical historical umbrella branch: `symphony/BIG-GOM-302`
- Canonical merged PR: `https://github.com/OpenAGIs/BigClaw/pull/138`
- Historical intermediate closeout PR: `https://github.com/OpenAGIs/BigClaw/pull/141`
- Recommendation: treat PR `#138` and the merged `main` history as the only
  merge source of truth for the completed cutover.
- Recommendation: do not reopen, cherry-pick, or independently merge old
  `BIG-GOM-*` slice branches after March 21, 2026. Any compatibility or
  documentation follow-up should branch from current `main`.

## Executable landing path

1. Start from current `main`, not from an archived `BIG-GOM-*` branch.
2. Use `docs/go-mainline-cutover-handoff.md`,
   `docs/go-mainline-cutover-issue-pack.md`, `docs/parallel-refill-queue.md`,
   and `local-issues.json` as the historical source set for what already
   landed.
3. Confirm the repo still matches the documented Go-first posture:
   `bigclaw-go` owns new implementation work and `src/bigclaw` is limited to
   migration-only compatibility shims.
4. Re-run the branch-complete validation commands below before any new
   compatibility landing or release cut.
5. Land new follow-up work only as forward-only changes from `main`, keeping
   distributed durability and live-validation caveats in the existing
   `BIG-PAR-*` / `OPE-*` follow-up tracks instead of reopening the cutover
   slices.

## First implementation and adaptation batch

Batch A should be the first landing group if the merged cutover posture needs
to be re-verified or adapted in a fresh branch:

- `BIG-GOM-301` through `BIG-GOM-303`
- Goal: lock the canonical Go contracts, risk/policy semantics, scheduler, and
  worker runtime before touching operator-facing surfaces.
- Primary regression surface:
  - `cd bigclaw-go && go test ./internal/domain ./internal/intake ./internal/workflow ./internal/scheduler ./internal/worker ./internal/queue ./internal/governance ./internal/contract ./internal/observability`

Batch B is the first adaptation wave on top of that runtime baseline:

- `BIG-GOM-304` through `BIG-GOM-306`
- Goal: confirm reporting, control-center, triage, and repo-lineage surfaces
  still reflect the Go-owned runtime data model.
- Primary regression surface:
  - `cd bigclaw-go && go test ./internal/reporting ./internal/regression ./internal/api ./internal/product ./internal/triage`

Batch C is the final compatibility and release gate:

- `BIG-GOM-307` through `BIG-GOM-309`
- Goal: confirm operator tooling, frozen Python shims, Git sync, and branch-wide
  validation evidence remain consistent with the Go-first posture.
- Primary regression surface:
  - `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/bootstrap ./internal/legacyshim`
  - `bash scripts/ops/bigclawctl legacy-python compile-check --json`
  - `PYTHONPATH=src python3 -m pytest -q tests/test_legacy_shim.py`

## Required validation commands

Run these commands in order for any mainline compatibility landing or release
checkpoint:

1. `cd bigclaw-go && go test ./...`
2. `bash scripts/ops/bigclawctl legacy-python compile-check --json`
3. `PYTHONPATH=src python3 -m pytest -q tests/test_legacy_shim.py`
4. `bash scripts/ops/bigclawctl github-sync status --json`
5. `bash scripts/ops/bigclawctl refill --local-issues local-issues.json`

Use the parallel validation matrix when the change touches distributed
execution, live validation, or rollout readiness:

- `cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=1 BIGCLAW_E2E_RUN_KUBERNETES=0 BIGCLAW_E2E_RUN_RAY=0 ./scripts/e2e/run_all.sh`
- `cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=0 BIGCLAW_E2E_RUN_KUBERNETES=1 BIGCLAW_E2E_RUN_RAY=0 ./scripts/e2e/run_all.sh`
- `cd bigclaw-go && BIGCLAW_E2E_RUN_LOCAL=0 BIGCLAW_E2E_RUN_KUBERNETES=0 BIGCLAW_E2E_RUN_RAY=1 ./scripts/e2e/run_all.sh`

Use `docs/reports/parallel-validation-matrix.md` and
`docs/reports/parallel-follow-up-index.md` as the discovery entrypoints for
lane-specific evidence and remaining follow-up caveats.

## Regression surface

The merge path must keep these surfaces aligned:

- contract and runtime packages under `bigclaw-go/internal/domain`,
  `internal/intake`, `internal/workflow`, `internal/scheduler`, and
  `internal/worker`
- operator and reviewer surfaces under `bigclaw-go/internal/reporting`,
  `internal/api`, `internal/product`, and `internal/regression`
- frozen Python compatibility shims under `src/bigclaw` plus the Go wrapper
  command `scripts/ops/bigclawctl legacy-python compile-check`
- repo-native status artifacts:
  `docs/go-mainline-cutover-handoff.md`,
  `docs/go-mainline-cutover-issue-pack.md`,
  `docs/parallel-refill-queue.md`,
  `docs/parallel-refill-queue.json`, and `local-issues.json`

## Branch and PR recommendation

- Use `symphony/BIG-GO-910` as the coordination branch for this planning slice.
- If a new PR is required, open exactly one forward-only PR from current `main`.
- Recommended PR title:
  - `BIG-GO-910: publish Go mainline merge and compatibility landing plan`
- PR body should link:
  - historical merged PR `#138`
  - `docs/go-mainline-cutover-handoff.md`
  - `docs/go-mainline-cutover-issue-pack.md`
  - `docs/reports/parallel-validation-matrix.md`
  - `docs/reports/parallel-follow-up-index.md`

## Main risks and controls

- Risk: stale branch resurrection or cherry-picks can reintroduce already-merged
  divergences.
  - Control: branch only from current `main`; treat PR `#138` as immutable
    history.
- Risk: Go-mainline confidence degrades if shim checks are skipped.
  - Control: keep `legacy-python compile-check` and `tests/test_legacy_shim.py`
    in the mandatory gate.
- Risk: doc-only merge confidence hides distributed-runtime caveats.
  - Control: require the parallel validation matrix for changes that touch live
    execution, takeover, continuation, or durability behavior.
- Risk: tracker and queue docs drift from the merged state.
  - Control: update `local-issues.json`, `docs/parallel-refill-queue.json`, and
    `docs/parallel-refill-queue.md` together whenever a future closeout slice
    changes the canonical status story.
