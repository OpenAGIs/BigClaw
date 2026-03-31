# BigClaw v5.3 Go Mainline Cutover Issue Pack

This document is the repo-native planning pack for the Go-only mainline cutover.

## Current Linear status

- Project created: `BigClaw v5.3 Go Mainline Consolidation`
- Project URL: `https://linear.app/openagi/project/bigclaw-v53-go-mainline-consolidation-53e33900c67e`
- Intended epic: `[Epic] BigClaw Go Mainline Cutover and Python Migration`
- Issue creation remains blocked as of March 18, 2026.
  - Verified by attempting to create the epic issue in Linear and receiving: `Usage limit exceeded - You've exceeded the free issue limit for this workspace.`
  - The project itself exists, but individual issue creation is still unavailable in the current workspace plan.
  - Re-verified later on March 18, 2026 by attempting to create six prepared migration slice issues; all six failed with the same workspace issue-limit error.

Until the Linear workspace issue limit is raised or old issue capacity is reclaimed, treat this file and the project status updates as the canonical issue-pack fallback for the Go-mainline project.

As of March 18, 2026, the execution tracker for this cutover is the
repo-native local issue store at `BigClaw/local-issues.json`; Linear metadata is
retained only as historical/project-context reference until workspace capacity
changes.

As of March 20, 2026, all planned local cutover slices `BIG-GOM-301` through
`BIG-GOM-308` are recorded as `Done` in `local-issues.json`, and the canonical
refill queue in `docs/parallel-refill-queue.json` is drained with no runnable
candidates remaining.

## March 18 tranche update

- `bigclaw-go/internal/intake/*`, `bigclaw-go/internal/workflow/*`, and `docs/go-domain-intake-parity-matrix.md` now own the first Go-side domain/intake/workflow parity slice from Python.
- `bigclaw-go/cmd/bigclawctl`, `bigclaw-go/internal/bootstrap/*`, `bigclaw-go/internal/githubsync/*`, and `bigclaw-go/internal/refill/*` now back the Go-first bootstrap / sync / refill operator toolchain.
- `bigclaw-go/internal/governance/freeze.go` now owns the migrated scope-freeze governance contract with round-trip coverage.
- `bigclaw-go/internal/contract/execution.go` now owns the migrated execution-contract and permission-matrix surface with Go parity tests.
- `bigclaw-go/internal/observability/audit_spec.go` now owns the canonical operational audit-event specification slice from Python.
- JSON default restoration for governance backlog items, execution fields, and audit policies now matches the active Go tests.
- Verification on March 18, 2026: `cd BigClaw/bigclaw-go && go test ./...` passed.

## March 20 completion update

- `BIG-GOM-301` through `BIG-GOM-308` are complete in the repo-native tracker.
- `bigclaw-go` owns the active mainline surfaces covered by the cutover issue pack.
- The remaining Python runtime entrypoints are explicitly frozen as migration-only compatibility paths.
- `bash scripts/ops/bigclawctl refill --local-issues local-issues.json` now reports no `In Progress` work and no refill candidates for this cutover set.

## Mainline policy

- `BigClaw/bigclaw-go` is the sole implementation mainline for new development.
- `BigClaw/src/bigclaw` should only be touched to migrate required surfaces to Go or to mark legacy Python paths as frozen/deprecated.
- Do not port historical Python helpers blindly; only migrate the surfaces required for active workflow, operator, reporting, and release paths.
- Keep `2-4` slices runnable in parallel once issue creation is available again.

## Milestones

### Mainline Declaration & Repo Routing

- declare `bigclaw-go` as the only implementation mainline
- route workflow/docs/contributor guidance to Go-first surfaces
- establish the canonical refill order for the cutover program

### Control/Workflow Surface Migration

- migrate the remaining Python-owned domain, policy, scheduler, workflow, and runtime-adjacent surfaces needed for Go-first execution

### Governance/Reporting Surface Migration

- move governance, reporting, control-center, triage, and repo-lineage surfaces needed for operator/reviewer workflows to Go-owned implementations

### Python Retirement & Cutover Validation

- freeze Python runtime/tooling paths
- validate parity and cutover safety
- finish the Go-only workflow and release switchover

## Draft epic

### [Epic] BigClaw Go Mainline Cutover and Python Migration

Goal:
Make `BigClaw/bigclaw-go` the sole implementation mainline and complete the staged migration of the remaining Python-owned mainline surfaces into Go.

Success criteria:

- `bigclaw-go` is the only documented implementation mainline
- workflow/docs/refill routing point contributors and automation at Go-first surfaces
- required Python-owned mainline surfaces have Go owners or are explicitly marked legacy
- cutover validation demonstrates that Go can replace the Python mainline for active development

## Archived issue slices

The sections below preserve the original slice definitions used to execute the
cutover. Any "initial state" value is historical and should be read as the
starting state that was used when the local tracker was first populated, not as
current runnable work.

### BIG-GOM-301 Unified domain model and intake contract migration

Python source:
- `src/bigclaw/models.py`

Go ownership:
- `bigclaw-go/internal/domain/task.go`
- `bigclaw-go/internal/domain/priority.go`
- `bigclaw-go/internal/intake/*`
- `bigclaw-go/internal/workflow/definition.go`
- `bigclaw-go/internal/workflow/model.go`
- `bigclaw-go/internal/prd/intake.go`

Acceptance focus:
- define the canonical Go task/intake/workflow vocabulary for all later migration slices
- produce a parity matrix for Python fields vs Go fields
- block later slices from inventing divergent contracts

Current repo progress:
- `bigclaw-go/internal/intake/*` now backs the active Go-first intake connector and source-issue mapping API surface
- `bigclaw-go/internal/workflow/definition.go` and `bigclaw-go/internal/workflow/model.go` now back the active Go workflow-definition and flow-contract surface
- `bigclaw-go/internal/risk/assessment.go` and `bigclaw-go/internal/triage/record.go` now own the migrated Python assessment / triage contract surface
- `bigclaw-go/internal/billing/statement.go` remains the canonical Go billing contract, with parity coverage expanded to preserve Python usage metadata during round trips
- `/v2/intake/connectors/...`, `/v2/intake/issues/map`, and `/v2/workflows/definitions/render` now expose Go-owned intake / mapping / workflow-definition endpoints for downstream tooling
- remaining `models.py` contract structs still need to be folded into the existing Go runtime / orchestration packages instead of copied into one compatibility file

Milestone:
- `Control/Workflow Surface Migration`

Historical initial state:
- `In Progress`

### BIG-GOM-302 Risk, policy, and approval semantics migration

Python source:
- `src/bigclaw/risk.py`
- `src/bigclaw/governance.py`
- `src/bigclaw/execution_contract.py`
- `src/bigclaw/audit_events.py`

Go ownership:
- `bigclaw-go/internal/risk/risk.go`
- `bigclaw-go/internal/policy/policy.go`
- `bigclaw-go/internal/api/policy_runtime.go`
- `bigclaw-go/internal/observability/audit.go`

Acceptance focus:
- port risk, approval, policy, and audit semantics required by the Go mainline
- align policy runtime and audit payloads with the canonical Go domain shape

Current repo progress:
- `bigclaw-go/internal/governance/freeze.go` now owns the Go scope-freeze backlog board and governance audit surface migrated from `src/bigclaw/governance.py`
- `bigclaw-go/internal/contract/execution.go` now owns the Go execution contract, permission matrix, and operations API contract migrated from `src/bigclaw/execution_contract.py`
- `bigclaw-go/internal/observability/audit_spec.go` now owns the canonical P0 audit event spec registry migrated from `src/bigclaw/audit_events.py`
- targeted Go tests for governance / contract / observability now pass, and `cd BigClaw/bigclaw-go && go test ./...` passed after this tranche
- Python source files remain in place as migration references; BigClaw is still not 100% Go

Dependencies:
- depends on `BIG-GOM-301`

Milestone:
- `Control/Workflow Surface Migration`

Historical initial state:
- `In Progress`

### BIG-GOM-303 Workflow orchestration and scheduler loop migration

Python source:
- `src/bigclaw/runtime.py`
- `src/bigclaw/scheduler.py`
- `src/bigclaw/orchestration.py`
- `src/bigclaw/workflow.py`
- `src/bigclaw/queue.py`

Go ownership:
- `bigclaw-go/internal/scheduler/scheduler.go`
- `bigclaw-go/internal/worker/runtime.go`
- `bigclaw-go/internal/orchestrator/loop.go`
- `bigclaw-go/internal/queue/queue.go`
- `bigclaw-go/internal/control/controller.go`

Acceptance focus:
- close the core Go execution loop for workflow-driven operation
- make Python execution kernel paths unnecessary for active development

Dependencies:
- depends on `BIG-GOM-301`
- depends on `BIG-GOM-302`

Milestone:
- `Control/Workflow Surface Migration`

Historical initial state:
- `Todo`

### BIG-GOM-304 Observability, reporting, and weekly operations surface migration

Python source:
- `src/bigclaw/observability.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/operations.py`

Go ownership:
- `bigclaw-go/internal/observability/recorder.go`
- `bigclaw-go/internal/observability/audit.go`
- `bigclaw-go/internal/reporting/reporting.go`
- `bigclaw-go/internal/regression/regression.go`

Acceptance focus:
- port the reporting and observability surfaces required for runtime closeout and reviewer evidence
- keep data shapes consistent with the Go runtime outputs

Dependencies:
- depends on `BIG-GOM-301`
- depends on `BIG-GOM-303`

Milestone:
- `Governance/Reporting Surface Migration`

Historical initial state:
- `Todo`

### BIG-GOM-305 Control center, triage, and operations view migration

Python source:
- `src/bigclaw/run_detail.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/saved_views.py`

Go ownership:
- `bigclaw-go/internal/api/server.go`
- `bigclaw-go/internal/api/v2.go`
- `bigclaw-go/internal/triage/triage.go`
- `bigclaw-go/internal/product/console.go`

Acceptance focus:
- move operator-facing status, control-center, and triage surfaces to Go-owned endpoints

Dependencies:
- depends on `BIG-GOM-303`
- depends on `BIG-GOM-304`

Milestone:
- `Governance/Reporting Surface Migration`

Historical initial state:
- `Backlog`

### BIG-GOM-306 Repo collaboration and lineage surface migration

Python source:
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_gateway.py`
- `src/bigclaw/repo_plane.py`
- `src/bigclaw/repo_board.py`

Go ownership:
- `bigclaw-go/internal/api/v2.go`
- `bigclaw-go/internal/reporting/reporting.go`
- `bigclaw-go/internal/flow/flow.go`
- optional new package family: `bigclaw-go/internal/repo/*`

Acceptance focus:
- move repo collaboration, lineage, and review surfaces required by the Go mainline
- keep repo-specific concerns out of the core scheduler/worker packages

Dependencies:
- depends on `BIG-GOM-301`
- depends on `BIG-GOM-304`

Current repo progress:
- `bigclaw-go/internal/repo/governance.go` now ports `src/bigclaw/repo_governance.py` into a Go-owned repo permission matrix and audit-field contract
- the remaining repo-collaboration Python surfaces still need Go owners across repo board, registry, gateway, plane, links, commits, and triage packages

Milestone:
- `Governance/Reporting Surface Migration`

Historical initial state:
- `Backlog`

### BIG-GOM-307 Workflow, bootstrap, and GitHub sync toolchain migration

Python source:
- `src/bigclaw/workspace_bootstrap.py`
- `scripts/ops/bigclaw_github_sync.py`
- `scripts/ops/symphony_workspace_bootstrap.py`
- `scripts/ops/bigclaw_refill_queue.py`

Go ownership:
- new `cmd/bigclawctl`
- new `bigclaw-go/internal/bootstrap/*`
- new `bigclaw-go/internal/githubsync/*`
- new `bigclaw-go/internal/refill/*`

Acceptance focus:
- replace Python workflow/bootstrap/refill helpers needed for Go-only operation
- keep the same shared-mirror and GitHub sync guarantees as the current workflow

Dependencies:
- depends on `BIG-GOM-303`

Current repo progress:
- `scripts/ops/bigclawctl` now routes operators into the Go `cmd/bigclawctl` entrypoint instead of the legacy Python helpers
- `bigclaw-go/internal/bootstrap/*` now owns shared-mirror bootstrap, cleanup, and validation logic with Go tests
- `bigclaw-go/internal/githubsync/*` now owns GitHub sync install / inspect / push guarantees with Go tests and hook integration
- `bigclaw-go/internal/refill/*` now owns the draft refill queue selection logic with tracker-neutral `TrackedIssue` records, while `cmd/bigclawctl refill` handles backend-specific polling and promotion
- `workflow.md`, `.githooks/post-commit`, and `.githooks/post-rewrite` now invoke the Go-first toolchain by default while legacy Python wrappers remain as compatibility shims
- at the time this slice was defined, the remaining Python wrappers still existed as migration shims and `BIG-GOM-308` was the planned follow-on slice to remove Python from the default operator path

Milestone:
- `Python Retirement & Cutover Validation`

Historical initial state:
- `In Progress`

### BIG-GOM-308 Python deprecation and Go-only mainline switch

Python source:
- `src/bigclaw/service.py`
- any remaining active Python entrypoints not covered by earlier slices

Go ownership:
- `bigclaw-go/cmd/bigclawd/main.go`
- `bigclaw-go/internal/api/server.go`
- new `cmd/bigclawctl`

Acceptance focus:
- remove Python from the default developer and runtime path
- leave legacy markers or archival notes where Python code is intentionally retained
- finish cutover validation and release-readiness evidence

Dependencies:
- depends on `BIG-GOM-301` through `BIG-GOM-307`

Milestone:
- `Python Retirement & Cutover Validation`

Historical initial state:
- `Backlog`

## Parallel execution order

Phase 1:
- `BIG-GOM-301`
- `BIG-GOM-302`

Phase 2:
- `BIG-GOM-303`

Phase 3:
- `BIG-GOM-304`
- `BIG-GOM-305`
- `BIG-GOM-306`

Phase 4:
- `BIG-GOM-307`

Phase 5:
- `BIG-GOM-308`

## First runnable batch once issue creation is available

- Historical initial batch: `BIG-GOM-301` and `BIG-GOM-302` entered `In Progress` first, followed by downstream refill activation from the canonical queue.

## Final local issue states as of March 20, 2026

- `BIG-GOM-301` -> `Done`
- `BIG-GOM-302` -> `Done`
- `BIG-GOM-303` -> `Done`
- `BIG-GOM-304` -> `Done`
- `BIG-GOM-305` -> `Done`
- `BIG-GOM-306` -> `Done`
- `BIG-GOM-307` -> `Done`
- `BIG-GOM-308` -> `Done`

## Archived refill batch blocked by Linear issue limits

These were the six concrete issue drafts queued for creation on March 18, 2026
before Linear rejected all `save_issue` attempts due to the workspace issue
limit. They are retained here as historical planning context rather than active
tracker work.

### 1. Close risk and policy parity on the Go mainline

Python source:
- `src/bigclaw/risk.py`
- remaining active consumers of `src/bigclaw/governance.py`
- remaining active consumers of `src/bigclaw/execution_contract.py`
- remaining active consumers of `src/bigclaw/audit_events.py`

Go ownership:
- `bigclaw-go/internal/risk/risk.go`
- `bigclaw-go/internal/policy/policy.go`
- `bigclaw-go/internal/api/policy_runtime.go`
- `bigclaw-go/internal/observability/audit.go`

Historical planned state:
- `Todo`

### 2. Port the workflow, scheduler, runtime, and orchestration loop to Go

Python source:
- `src/bigclaw/runtime.py`
- `src/bigclaw/scheduler.py`
- `src/bigclaw/orchestration.py`
- `src/bigclaw/workflow.py`
- `src/bigclaw/queue.py`

Go ownership:
- `bigclaw-go/internal/worker/runtime.go`
- `bigclaw-go/internal/scheduler/*`
- `bigclaw-go/internal/orchestrator/loop.go`
- `bigclaw-go/internal/control/controller.go`
- `bigclaw-go/internal/queue/*`

Historical planned state:
- `Todo`

### 3. Port observability, reports, and operations evidence surfaces to Go

Python source:
- `src/bigclaw/observability.py`
- `src/bigclaw/reports.py`
- `src/bigclaw/operations.py`
- `src/bigclaw/evaluation.py`
- `src/bigclaw/run_detail.py`
- `src/bigclaw/planning.py`

Go ownership:
- `bigclaw-go/internal/observability/*`
- `bigclaw-go/internal/reporting/reporting.go`
- `bigclaw-go/internal/regression/regression.go`
- `bigclaw-go/internal/api/server.go`
- `bigclaw-go/internal/triage/*`
- `bigclaw-go/internal/billing/*`

Historical planned state:
- `Todo`

### 4. Port repo collaboration and lineage surfaces to Go

Python source:
- `src/bigclaw/collaboration.py`
- `src/bigclaw/repo_board.py`
- `src/bigclaw/repo_links.py`
- `src/bigclaw/repo_plane.py`

Go ownership:
- `bigclaw-go/internal/flow/flow.go`
- `bigclaw-go/internal/githubsync/*`
- `bigclaw-go/internal/triage/*`
- optional new `bigclaw-go/internal/repo/*`
- `bigclaw-go/internal/product/console.go`

Historical planned state:
- `Backlog`

### 5. Port operator console and saved-view surfaces to Go

Python source:
- `src/bigclaw/console_ia.py`
- `src/bigclaw/design_system.py`
- `src/bigclaw/saved_views.py`
- `src/bigclaw/ui_review.py`
- remaining operator-facing parts of `src/bigclaw/service.py`

Go ownership:
- `bigclaw-go/internal/product/console.go`
- `bigclaw-go/internal/api/v2.go`
- `bigclaw-go/internal/api/server.go`
- optional new `bigclaw-go/internal/product/views.go`

Historical planned state:
- `Backlog`

### 6. Replace Python bootstrap and sync entrypoints with Go-only tooling

Python source:
- `src/bigclaw/workspace_bootstrap.py`
- `src/bigclaw/service.py`

Go ownership:
- `bigclaw-go/cmd/bigclawctl`
- `bigclaw-go/cmd/bigclawd`
- `bigclaw-go/internal/bootstrap/*`
- `bigclaw-go/internal/githubsync/*`
- `bigclaw-go/internal/refill/*`
- `bigclaw-go/internal/api/server.go`

Historical planned state:
- `Backlog`
