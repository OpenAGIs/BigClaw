# BigClaw v5.0 Operations Foundation Evidence Pack

Date: 2026-03-16

## Scope

This report refreshes the legacy operations-foundation summary into a current v5.0 evidence pack for the distributed control plane implemented in `bigclaw-go`.

The pack is intentionally limited to repo-native surfaces that are implemented and testable today:
- control-center operations and audit
- scheduler policy inspection and reload
- run detail, audit, and report drilldowns
- engineering, operations, triage, and regression review dashboards
- collaboration and takeover context carried through control-center and run-detail responses

## Reviewer-facing control-plane surfaces

### Dashboard and review surfaces
- `GET /v2/dashboard/engineering`
  - Engineering summary with team, project, tenant, and time-window filters
  - Ticket-to-merge funnel, blocker counts, premium usage, budget totals, and per-task drilldowns
- `GET /v2/dashboard/operations`
  - Operations summary with active, blocked, overdue, and SLA-risk run visibility
  - Project/team breakdowns plus hourly or daily trend views
- `GET /v2/triage/center`
  - Risk-ranked triage inbox with suggested next actions, owners, workflows, and similar-case context
- `GET /v2/regression/center`
  - Regression hotspot view with workflow, template, service, and compare-window summaries

### Control-center and collaboration surfaces
- `GET /v2/control-center`
  - Control-plane pause state, event-log capability summary, queue inventory, dead letters, worker-pool packaging, distributed diagnostics, and recent task snapshots
  - Queue breakdowns by team/project plus active takeovers, recent control actions, notes timeline, and optional checkpoint reset audit visibility
- `GET /v2/control-center/audit`
  - Filterable control-action history by task, actor, owner, reviewer, team, project, action, and scope
  - Aggregated audit facets and a reviewer-facing notes timeline
- `POST /v2/control-center/actions`
  - Operational actions for `pause`, `resume`, `retry`, `cancel`, `takeover`, `transfer_to_human`, `release_takeover`, `annotate`, `assign_owner`, and `assign_reviewer`
  - Role-aware authorization and normalized operation payloads for audit/event reuse

### Scheduler policy surfaces
- `GET /v2/control-center/policy`
  - Current scheduler policy snapshot, fairness state, storage backend metadata, and reload capability
- `POST /v2/control-center/policy/reload`
  - Source-backed scheduler policy reload for authorized roles with updated fairness/policy payloads

### Run detail surfaces
- `GET /v2/runs/{task_id}`
  - Task snapshot, policy and risk summary, collaboration context, trace summary, lifecycle timeline, validation status, tool traces, artifact refs, workpad metadata, and report links
- `GET /v2/runs/{task_id}/audit`
  - Run-scoped control-action audit stream with the same normalized entry shape used by control-center audit
- `GET /v2/runs/{task_id}/report`
  - Markdown-ready run report summarizing task state, policy, collaboration notes, validation, artifacts, and recent actions

## Supporting evidence and terminology alignment

- Observability and debug evidence for metrics, traces, worker status, and JSONL audit persistence lives in `docs/reports/go-control-plane-observability-report.md`.
- Review-pack coverage across the rewrite baseline and follow-up hardening lives in `docs/reports/review-readiness.md`.
- The current API registration for the control-plane surfaces above lives in `internal/api/server.go`.
- Handler implementations for the dashboard, control-center, and run-detail payloads live in `internal/api/v2.go`.
- Scheduler policy inspection and reload handlers live in `internal/api/policy_runtime.go`.
- Collaboration/takeover state is backed by `internal/control/controller.go`.
- Task snapshots, traces, and audit data are recorder-backed via `internal/observability/recorder.go` and `internal/observability/audit.go`.

## Current posture and non-goals

- This pack reflects the current repo-native v5.0 distributed-platform control plane. It does not claim external telemetry backends, production leader election, or higher-scale external-store certification.
- Event-log and distributed diagnostics surfaces are implementation-facing summaries for operator review, not a claim of production-grade multi-region rollout.
- Collaboration coverage is limited to implemented takeover, ownership, reviewer, and note timelines already exposed through the control-center and run-detail APIs.

## Validation

Commands run:
- `go test ./internal/api`

Results:
- `ok  	bigclaw-go/internal/api`
- Report consistency coverage verifies that the evidence pack references currently registered control-plane endpoints and that `docs/reports/review-readiness.md` links back to this refreshed report.
