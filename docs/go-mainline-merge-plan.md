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

## Current parallel branch inventory as of March 27, 2026

These are the follow-up branches visible from `origin` while preparing
`BIG-GO-910`:

| Branch | Visible head | Scope signal | Recommended merge posture |
| --- | --- | --- | --- |
| `origin/codex/big-go-901-migration-inventory` | `c7fc2049e0e03909017fddb3da7b257abf65bab4` | migration inventory and ownership baseline | merge first so later slices inherit the same source-of-truth inventory |
| `origin/feat/BIG-GO-902-go-cli-script-migration` | `70cddf84fdb07bdf31d12c52c91e167a4ab8ab28` | Go CLI and script migration surface | merge after the inventory baseline and before operator/bootstrap follow-ups |
| `origin/BIG-GO-903` | `365b6ca167b3bd50c76198d924c979a8d1d3f115` | test harness migration plan and doc/regression guardrails | merge before runtime and closeout slices so the Go validation gate is shared |
| `origin/codex/BIG-GO-904-control-plane-go-only-slice` | `da40cb7c169576145fb9d77050dfe12fd6385439` | control-plane Go-only migration slice | merge before orchestration/tooling follow-ups because it changes the mainline ownership boundary |
| `origin/big-go-905` | `0f3b0394a6bac69a15ecda803e5f2dc82da9c02b` | repo governance and board capability migration | merge after the core control-plane/runtime story is stable |
| `origin/codex/BIG-GO-906-runtime-scheduler-orchestration-migration` | `f6cb4b6325fe9167798b71f47d507b8f9c230410` | runtime/scheduler/orchestration migration plan | merge early because later operator/reporting surfaces depend on the runtime ownership story |
| `origin/symphony/BIG-GO-908` | `54aaffaec53d26d1d81e192df8794699075c140b` | workspace bootstrap lifecycle migration to Go CLI | merge after the CLI/control-plane slices; low overlap outside bootstrap docs and tooling |
| `origin/symphony/BIG-GO-909` | `d38ce57494ef7c689476aedef7ba1c72980a2d83` | repo collaboration and GitHub-sync follow-up | merge after `BIG-GO-905` and `BIG-GO-908` so repo metadata and sync tooling converge once |
| `origin/symphony/BIG-GO-910` | `a63894abcc590c1c6518cafb661dc50ef489243c` | parallel closeout and main merge plan | keep last; it should close over the validated outputs of the earlier slices |

`BIG-GO-907` was not visible from `origin` in this workspace on March 27, 2026.
Treat that as a required resync or branch-restoration check before claiming
that all expected parallel inputs are ready for an umbrella closeout.

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
- Recommended sequencing for the currently visible follow-up branches:
  1. `BIG-GO-901`
  2. `BIG-GO-902`
  3. `BIG-GO-903`
  4. `BIG-GO-904`
  5. `BIG-GO-906`
  6. `BIG-GO-905`
  7. `BIG-GO-908`
  8. `BIG-GO-909`
  9. `BIG-GO-910`
- PR strategy:
  - keep one bounded PR per active `BIG-GO-*` branch
  - use `BIG-GO-910` only for the merge-plan, compatibility gate, and final
    closeout guidance
  - only open an umbrella execution PR after the visible branch heads above are
    rebased onto current `main` and their targeted validation commands are
    recorded in-branch

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
- Risk: `.github/workflows/ci.yml` currently runs Python lint/test/build only,
  so Go migration branches can look green without exercising the Go merge gate.
  - Control: record targeted `go test` commands in every `BIG-GO-*` PR and
    treat them as required reviewer evidence until CI adds Go-native coverage.
