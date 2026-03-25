# ClawHost Control-Plane Review Index

This document is the reviewer entrypoint for the checked-in ClawHost Go
control-plane tranche that landed through PR `#186` and merged into `main` on
March 24, 2026.

## Included slices

- `BIG-PAR-287` - ClawHost fleet inventory and control-plane source
- `BIG-PAR-288` - parallel ClawHost rollout planner
- `BIG-PAR-289` - ClawHost skills channels and device approval workflows
- `BIG-PAR-290` - ClawHost provider defaults and tenant policy surfaces
- `BIG-PAR-291` - ClawHost proxy subdomain and admin validation lane
- `BIG-PAR-292` - ClawHost lifecycle recovery and per-bot isolation scorecard
- `BIG-PAR-293` through `BIG-PAR-297` - focused and aggregate `/v2/control-center`
  regressions for the merged reviewer-facing ClawHost bundle

## Review order

1. Fleet inventory:
   `docs/reports/clawhost-fleet-inventory-surface.json`
2. Tenant policy and provider defaults:
   `docs/reports/clawhost-tenant-policy-surface.json`
3. Proxy and admin validation:
   `docs/reports/clawhost-proxy-admin-validation-lane.json`
4. Rollout planner:
   `docs/reports/clawhost-rollout-planner-surface.json`
5. Reviewer-facing workflow surface:
   `bigclaw-go/internal/product/clawhost_workflow.go`
6. Reviewer-facing readiness surface:
   `bigclaw-go/internal/api/clawhost_readiness_surface.go`
7. Reviewer-facing recovery scorecard:
   `bigclaw-go/internal/product/clawhost_recovery.go`
8. Aggregate API wiring and cross-surface regressions:
   `bigclaw-go/internal/api/server.go`
   `bigclaw-go/internal/api/server_test.go`

## Runtime entrypoints

- `/debug/status`
- `/v2/control-center`
- `/v2/control-center/policy`
- `/v2/reports/distributed`
- `/v2/clawhost/fleet`
- `/v2/clawhost/rollout-planner`
- `/v2/clawhost/workflows`
- `/v2/clawhost/recovery-scorecard`

## Artifact intent

- `clawhost-fleet-inventory-surface.json`
  - reviewer-facing inventory for app/bot ownership, lifecycle state, runtime region, and domains
- `clawhost-tenant-policy-surface.json`
  - tenant-level provider defaults, approval posture, and rollout guardrails
- `clawhost-proxy-admin-validation-lane.json`
  - HTTP, WebSocket, subdomain, and admin readiness evidence
- `clawhost-rollout-planner-surface.json`
  - canary, parallel-wave, rollback, and takeover planning evidence
- `internal/product/clawhost_workflow.go`
  - reviewer-facing workflow queue for skill sync, IM channel review, pairing approvals, and takeover-gated credential exposure
- `internal/api/clawhost_readiness_surface.go`
  - reviewer-facing readiness checks for proxy mode, gateway ports, websocket reachability, admin posture, and subdomain health
- `internal/product/clawhost_recovery.go`
  - lifecycle recovery scorecard with per-bot isolation, takeover triggers, and release-readiness auditing
- `internal/api/server_test.go`
  - focused and aggregate `/v2/control-center` regressions pinning the merged policy, workflow, rollout, readiness, and recovery bundle

## Branch evidence

- PR: `https://github.com/OpenAGIs/BigClaw/pull/186`
- PR state: `MERGED`
- Merge commit: `bd854cd1f53808d529d7ae7413f01320eeeef337`
- Source branch: `symphony/BIG-PAR-291`
- Sync audit command:
  `bash scripts/ops/bigclawctl github-sync status --json`

## Boundary

- These artifacts define the Go reviewer contract for ClawHost control-plane
  surfaces in this repo.
- They do not execute a live external ClawHost API crawl or perform live bot
  lifecycle mutations from this checkout.
