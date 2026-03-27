# Runtime, Scheduler, and Orchestration Migration Plan

## Scope

This plan captures the executable follow-on work for `BIG-GO-906`: retire the
remaining Python-owned `runtime` / `scheduler` / `orchestration` / `workflow`
test and script surfaces behind the Go mainline without widening scope into
unrelated product or reporting migrations.

## Source inventory

### Legacy Python sources still acting as migration references

- `src/bigclaw/runtime.py`
- `src/bigclaw/scheduler.py`
- `src/bigclaw/orchestration.py`
- `src/bigclaw/workflow.py`

### Python tests to retire behind Go coverage

| Python test | Current intent | Go ownership target | Exit condition |
| --- | --- | --- | --- |
| `tests/test_runtime.py` | sandbox routing, tool policy, worker lifecycle | `internal/worker/runtime_test.go` | Go tests cover sandbox/runner policy, blocked tools, approval wait, and paused-runtime closeout states. |
| `tests/test_scheduler.py` | risk and budget routing decisions | `internal/scheduler/scheduler_test.go` | Go tests assert the same executor/risk/budget outcomes and reasons without Python fallback. |
| `tests/test_orchestration.py` | cross-department planning, policy, and handoff emission | `internal/workflow/orchestration_test.go`, `internal/scheduler/scheduler_test.go` | Go tests cover plan composition, entitlement policy, handoff records, and emitted audit/event payloads. |
| `tests/test_workflow.py` | workpad journal, acceptance, orchestration artifacts, closeout | `internal/workflow/engine_test.go`, `internal/workflow/closeout_test.go`, `internal/worker/runtime_test.go` | Go tests persist workpad/closeout state and prove acceptance plus orchestration artifact handling end to end. |
| `tests/test_runtime_matrix.py` | combined runtime/scheduler policy matrix | `internal/worker/runtime_test.go`, `internal/scheduler/scheduler_test.go` | A Go matrix test covers multi-tool worker execution and routing outcomes across low/high/browser tasks. |
| `tests/test_execution_flow.py`, `tests/test_audit_events.py`, `tests/test_dsl.py`, `tests/test_evaluation.py`, `tests/test_risk.py`, `tests/test_operations.py` | scheduler/workflow side effects consumed by adjacent domains | `internal/scheduler/*`, `internal/workflow/*`, `internal/observability/*`, `internal/regression/*` | Each Python assertion has a named Go owner before the Python test is deleted. |

### Python scripts still coupled to runtime migration proof

| Python script | Current role | Go ownership target | Priority |
| --- | --- | --- | --- |
| `bigclaw-go/scripts/e2e/run_task_smoke.py` | submit/poll helper for runtime executor smoke runs | `cmd/bigclawctl` subcommand or `scripts/e2e/run_task_smoke.go` | P0 |
| `bigclaw-go/scripts/e2e/export_validation_bundle.py` | export runtime validation bundle/index | `internal/regression` library plus `scripts/e2e/export_validation_bundle.go` | P0 |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py` | summarize validation lineage after runtime runs | `internal/regression` library plus Go CLI wrapper | P1 |
| `bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py` | evaluate continuation gate for runtime validation bundles | `internal/regression` library plus Go CLI wrapper | P1 |
| `bigclaw-go/scripts/e2e/multi_node_shared_queue.py` | shared runtime queue and takeover proof | keep Python harness first, then extract stable logic into Go test helper and thin launcher | P1 |
| `bigclaw-go/scripts/e2e/cross_process_coordination_surface.py` | runtime capability surface generation | move generation into `internal/regression` with a thin Go exporter | P1 |
| `bigclaw-go/scripts/migration/shadow_compare.py` | compare incumbent vs Go runtime outcomes | keep until live shadow path is replaced; then move compare logic into Go library | P1 |
| `bigclaw-go/scripts/migration/shadow_matrix.py` | batch migration parity matrix | same as above | P1 |
| `bigclaw-go/scripts/migration/live_shadow_scorecard.py` | score live shadow drift/readiness | `internal/regression` plus Go exporter | P1 |
| `bigclaw-go/scripts/migration/export_live_shadow_bundle.py` | export live shadow bundle/index | `internal/regression` plus Go exporter | P1 |

## Go ownership boundary

- `internal/scheduler/scheduler.go` owns route selection, budget policy,
  fairness, and scheduling decisions.
- `internal/worker/runtime.go` owns lease-to-execution lifecycle, handoff
  payload emission, acceptance closeout integration, and runtime workpad
  persistence.
- `internal/workflow/orchestration.go` and `internal/workflow/engine.go` own
  orchestration policy, collaboration planning, and workflow closeout state.
- `internal/orchestrator/loop.go` owns the long-running orchestration loop.
- `internal/regression/*` owns repo-native report/export generation once Python
  wrappers are retired.

## Execution sequence

### Phase 1: lock parity targets before deleting Python references

1. Freeze the Python source files above as reference-only inputs.
2. Create a one-to-one mapping from each Python assertion to an existing or new
   Go test case.
3. Add doc/regression coverage so reviewer paths always point to the canonical
   Go owner and validation command set.

### Phase 2: port the core Python tests to Go

1. Port `tests/test_runtime.py` and `tests/test_runtime_matrix.py` into
   `internal/worker/runtime_test.go`.
2. Port `tests/test_scheduler.py` plus scheduler assertions embedded in the
   orchestration/runtime tests into `internal/scheduler/scheduler_test.go`.
3. Port `tests/test_orchestration.py` into
   `internal/workflow/orchestration_test.go` and keep scheduler handoff checks
   in `internal/scheduler/scheduler_test.go`.
4. Port `tests/test_workflow.py` coverage into `internal/workflow/engine_test.go`
   and `internal/workflow/closeout_test.go`.
5. Only remove each Python test after the matching Go test is green in the same
   branch and the behavior diff is documented in the PR.

### Phase 3: convert runtime-facing Python scripts into Go-owned exporters

1. Move shared JSON/report shaping logic from Python scripts into
   `internal/regression` packages first.
2. Replace `run_task_smoke.py` with a Go command path so executor smoke lanes no
   longer require Python for task submission and polling.
3. Replace validation-bundle and live-shadow exporters with Go binaries or
   `bigclawctl` subcommands, then keep shell wrappers stable.
4. Leave complex multi-node/live-provider harnesses in Python until the stable
   report schemas are owned by Go libraries; avoid rewriting harness plumbing
   and report generation in one PR.

## First implementation and retrofit batch

The first batch should stay narrow enough for one bounded migration tranche:

1. Port `tests/test_runtime.py`, `tests/test_scheduler.py`,
   `tests/test_orchestration.py`, `tests/test_workflow.py`, and
   `tests/test_runtime_matrix.py` into the existing Go test packages listed
   above.
2. Add missing Go assertions for:
   `paused` runtime closeout, approval wait states, orchestration entitlement
   downgrades, handoff emission, workpad journal replay, and acceptance closeout
   artifacts.
3. Keep the Python tests temporarily, run both suites in parallel, and delete
   Python coverage only after the Go suite proves identical operator-visible
   behavior.
4. Start script migration with `run_task_smoke.py` and
   `export_validation_bundle.py`; both are high leverage because they sit on the
   runtime validation happy path and unblock later wrapper cleanup.

## Validation commands

Use the Python suite to establish the legacy baseline before removing coverage:

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906 && PYTHONPATH=src python3 -m pytest tests/test_runtime.py tests/test_scheduler.py tests/test_orchestration.py tests/test_workflow.py tests/test_runtime_matrix.py -q`

Use the Go suite as the migration gate:

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906/bigclaw-go && go test ./internal/worker ./internal/scheduler ./internal/workflow ./internal/orchestrator -count=1`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906/bigclaw-go && go test ./internal/regression -run 'Test(RuntimeSchedulerOrchestrationMigrationPlanDocs|MigrationFollowUpIndexDocsStayAligned|PlanningFollowUpIndexDocsStayAligned)' -count=1`

Use the executor-lane validation bundle after script migration starts:

- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906/bigclaw-go && ./scripts/e2e/run_all.sh`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906/bigclaw-go && python3 scripts/migration/live_shadow_scorecard.py --pretty`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-906/bigclaw-go && python3 scripts/migration/export_live_shadow_bundle.py`

## Regression surface

Any runtime/scheduler/orchestration migration PR must review and rerun:

- `internal/worker/runtime_test.go`
- `internal/scheduler/scheduler_test.go`
- `internal/workflow/orchestration_test.go`
- `internal/workflow/engine_test.go`
- `internal/workflow/closeout_test.go`
- `internal/orchestrator/loop_test.go`
- `internal/regression/*migration*`
- `internal/regression/*runtime*`
- `docs/migration.md`
- `docs/reports/migration-readiness-report.md`
- `docs/reports/migration-plan-review-notes.md`
- `docs/reports/parallel-validation-matrix.md`

## Branch and PR recommendation

- Branch name: `codex/BIG-GO-906-runtime-scheduler-orchestration-migration`
- PR 1: planning and regression guardrails only
- PR 2: Python test parity ports into Go test packages
- PR 3: runtime validation/export script migration (`run_task_smoke`,
  validation-bundle exporters, live-shadow exporters)
- PR 4: delete Python test/script leftovers after two consecutive green runs of
  the Go path plus validation-bundle refresh

## Risks

- Runtime closeout behavior can drift silently if Python tests are deleted before
  Go tests cover `paused`, `needs-approval`, and handoff-only paths.
- Script rewrites can conflate transport changes with report-shape changes;
  move logic into Go libraries first and preserve JSON schemas.
- Live validation commands touch Kubernetes and Ray; keep them as post-port
  evidence, not as the only migration gate.
- The multi-node takeover and live-shadow harnesses have environment-sensitive
  behavior; keep the harness launcher thin until the underlying report schema is
  stable in Go.
