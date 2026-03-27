# Test Harness Migration Plan

## Scope

This plan breaks the legacy Python `pytest` primary test surface into three Go-native harness lanes for `BIG-GO-903`:

- `go test` package tests for domain, API, scheduler, queue, worker, and product logic
- golden/doc-alignment regression tests for checked-in reports, migration docs, and machine-readable review surfaces
- integration harness lanes for deterministic local proofs plus live local/Kubernetes/Ray validation bundles

The goal is not a single big-bang port. The target state is a thinner Python compatibility shell around a repo-native Go validation stack with explicit commands, artifacts, and rollback-aware reviewer entrypoints.

## Current Harness Split

| Harness lane | Legacy Python primary surface | Go-native target surface | Canonical command |
| --- | --- | --- | --- |
| Unit and domain logic | `tests/test_queue.py`, `tests/test_scheduler.py`, `tests/test_runtime.py`, `tests/test_workflow.py`, `tests/test_execution_contract.py`, `tests/test_repo_*`, `tests/test_governance.py`, `tests/test_memory.py` | `internal/domain/*_test.go`, `internal/queue/*_test.go`, `internal/scheduler/*_test.go`, `internal/worker/*_test.go`, `internal/workflow/*_test.go`, `internal/contract/execution_test.go`, `internal/repo/*_test.go`, `internal/governance/freeze_test.go`, `internal/refill/*_test.go` | `cd bigclaw-go && go test ./internal/... ./cmd/...` |
| Golden and doc/regression alignment | `tests/test_followup_digests.py`, `tests/test_live_shadow_bundle.py`, `tests/test_shadow_matrix_corpus.py`, `tests/test_cross_process_coordination_surface.py`, `tests/test_validation_bundle_continuation_scorecard.py`, `tests/test_validation_bundle_continuation_policy_gate.py` | `internal/regression/*.go` plus doc/report JSON fixtures under `docs/reports/` | `cd bigclaw-go && go test ./internal/regression` |
| Integration and executable harnesses | `tests/test_subscriber_takeover_harness.py`, `tests/test_live_shadow_scorecard.py`, `tests/test_parallel_validation_bundle.py`, selected legacy wrapper tests around migration scripts | `scripts/e2e/*.py`, `scripts/migration/*.py`, `scripts/e2e/*.sh`, and their checked-in report outputs | `cd bigclaw-go && ./scripts/e2e/run_all.sh`; targeted `python3 scripts/e2e/...` or `python3 scripts/migration/...` rerenders |

## Migration Lanes

### 1. Go package tests as the default logic harness

Use package-local `go test` as the default home for behavior that does not need filesystem-backed golden artifacts or multi-process orchestration.

Migration rule:

- Port assertion logic out of `tests/test_*.py` when the subject under test already exists in `bigclaw-go/internal/...`.
- Keep one package-local test file per behavior seam instead of one large cross-module Python suite.
- Prefer table-driven tests for protocol/state permutations and boundary checks.

Primary command set:

```bash
cd bigclaw-go
go test ./internal/domain ./internal/queue ./internal/scheduler ./internal/worker ./internal/workflow ./internal/contract ./internal/repo ./internal/governance ./internal/refill
```

### 2. Golden/doc alignment via `internal/regression`

Use `internal/regression` as the golden layer for checked-in reports, migration digests, doc indexes, and machine-readable review surfaces.

Migration rule:

- Any checked-in markdown or JSON artifact that gates migration review should have a Go regression test under `internal/regression`.
- Python tests may remain as temporary structure checks while the equivalent Go regression coverage lands.
- Generated evidence should be validated by stable substrings, JSON schema expectations, and cross-link presence rather than by brittle whole-file snapshots.

Primary command set:

```bash
cd bigclaw-go
go test ./internal/regression
```

### 3. Integration harnesses for deterministic and live proofs

Use integration harnesses only when package-local tests cannot prove the behavior:

- deterministic local proofs for takeover, broker-failover stubs, bundle continuation, and shadow bundle exports
- live local/Kubernetes/Ray smoke validation for executor wiring and reviewer bundles
- migration shadow comparison for parity, drift, and rollback review

Migration rule:

- The harness must emit a checked-in report or bundle path that a regression test can pin.
- The harness command and its artifact path must be documented in `docs/e2e-validation.md`, `docs/reports/parallel-validation-matrix.md`, or migration docs.
- Avoid adding new pytest-only wrappers when the harness can be rerun directly and validated through checked-in artifacts.

Primary command set:

```bash
cd bigclaw-go
python3 scripts/e2e/subscriber_takeover_fault_matrix.py --pretty
python3 scripts/e2e/cross_process_coordination_surface.py --pretty
python3 scripts/e2e/validation_bundle_continuation_scorecard.py --pretty
python3 scripts/migration/live_shadow_scorecard.py --pretty
./scripts/e2e/run_all.sh
```

## First-Batch Implementation and Retrofit List

### Batch A: immediate Go-first conversions

1. Move queue/scheduler/runtime acceptance deltas out of `tests/test_queue.py`, `tests/test_scheduler.py`, and `tests/test_runtime.py` into package-local assertions in `internal/queue/*_test.go`, `internal/scheduler/scheduler_test.go`, and `internal/worker/runtime_test.go`.
2. Treat workflow and execution-contract behavior as Go-owned by continuing the port from `tests/test_workflow.py` and `tests/test_execution_contract.py` into `internal/workflow/*_test.go` and `internal/contract/execution_test.go`.
3. Keep repo/governance surface parity under `internal/repo/*_test.go`, `internal/triage/*_test.go`, and `internal/governance/freeze_test.go` instead of adding new cross-module pytest coverage.

### Batch B: golden/doc-alignment consolidation

1. Continue replacing report-structure pytest checks from `tests/test_followup_digests.py`, `tests/test_live_shadow_bundle.py`, and `tests/test_cross_process_coordination_surface.py` with `internal/regression/*_test.go`.
2. Use `internal/regression/live_shadow_bundle_surface_test.go`, `internal/regression/cross_process_coordination_docs_test.go`, `internal/regression/live_validation_index_test.go`, and this issue's new regression test as the template for new doc/report gates.
3. Keep Python only for short-term bridge coverage where a Go-side parser or fixture helper has not been added yet.

### Batch C: integration harness normalization

1. Keep `scripts/e2e/subscriber_takeover_fault_matrix.py` and `scripts/e2e/multi_node_shared_queue.py` as the canonical takeover harness pair; treat `tests/test_subscriber_takeover_harness.py` as a temporary report-shape guard until the remaining report invariants are covered in `internal/regression`.
2. Keep `scripts/migration/live_shadow_scorecard.py`, `scripts/migration/shadow_matrix.py`, and `scripts/migration/export_live_shadow_bundle.py` as the canonical migration harness entrypoints; regression tests should pin their checked-in outputs instead of wrapping them with new pytest-only behavior.
3. Keep `scripts/e2e/run_all.sh`, `scripts/e2e/kubernetes_smoke.sh`, and `scripts/e2e/ray_smoke.sh` as the only live executor entrypoints; the reviewer-facing contract remains the exported JSON bundle and `docs/reports/parallel-validation-matrix.md`.

## Validation and Regression Surface

Minimum validation for a harness migration PR:

```bash
python3 -m pytest tests/test_harness_migration_plan.py
cd bigclaw-go && go test ./internal/regression
```

Use targeted package tests when a specific batch item changes:

```bash
cd bigclaw-go
go test ./internal/queue ./internal/scheduler ./internal/worker
go test ./internal/workflow ./internal/contract
go test ./internal/repo ./internal/governance ./internal/triage
```

Use integration rerenders only when touched artifacts or harness scripts change:

```bash
cd bigclaw-go
python3 scripts/e2e/subscriber_takeover_fault_matrix.py --pretty
python3 scripts/migration/live_shadow_scorecard.py --pretty
./scripts/e2e/run_all.sh
```

## Branch and PR Recommendation

- Branch naming: `BIG-GO-903-test-harness-migration`
- PR sequencing:
  - PR 1: doc/regression harness inventory and guardrails
  - PR 2: unit/domain test migrations by package family
  - PR 3: integration harness cleanup and legacy pytest wrapper removal
- Merge gate:
  - require `go test ./internal/regression`
  - require targeted `go test` package coverage for any migrated area
  - require checked-in artifact rerender only when harness outputs change

## Risks

- Python and Go tests can drift if both remain authoritative for the same behavior; each migrated behavior needs a single owner lane.
- Live executor harnesses are environment-dependent, so they must stay artifact-backed and not become a required per-PR developer gate.
- Doc/report regression coverage can become noisy if it snapshots too much text; prefer stable invariants, command strings, and cross-link guarantees.
- Some legacy Python surfaces still cover compatibility-only code in `src/bigclaw/`; remove those wrappers only after the corresponding migration-only path is explicitly retired.
