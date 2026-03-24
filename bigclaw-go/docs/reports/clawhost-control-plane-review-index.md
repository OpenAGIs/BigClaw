# ClawHost Control-Plane Review Index

This document is the reviewer entrypoint for the checked-in ClawHost Go
control-plane slice batch completed on branch `symphony/BIG-PAR-291`.

## Included slices

- `BIG-PAR-287` - ClawHost fleet inventory and control-plane source
- `BIG-PAR-288` - parallel ClawHost rollout planner
- `BIG-PAR-289` - ClawHost skills channels and device approval workflows
- `BIG-PAR-290` - ClawHost provider defaults and tenant policy surfaces
- `BIG-PAR-291` - ClawHost proxy subdomain and admin validation lane

## Review order

1. Fleet inventory:
   `docs/reports/clawhost-fleet-inventory-surface.json`
2. Tenant policy and provider defaults:
   `docs/reports/clawhost-tenant-policy-surface.json`
3. Proxy and admin validation:
   `docs/reports/clawhost-proxy-admin-validation-lane.json`
4. Rollout planner:
   `docs/reports/clawhost-rollout-planner-surface.json`
5. Workflow contract:
   `internal/workflow/clawhost_workflows.go`

## Runtime entrypoints

- `/debug/status`
- `/v2/control-center`
- `/v2/reports/distributed`

## Artifact intent

- `clawhost-fleet-inventory-surface.json`
  - reviewer-facing inventory for app/bot ownership, lifecycle state, runtime region, and domains
- `clawhost-tenant-policy-surface.json`
  - tenant-level provider defaults, approval posture, and rollout guardrails
- `clawhost-proxy-admin-validation-lane.json`
  - HTTP, WebSocket, subdomain, and admin readiness evidence
- `clawhost-rollout-planner-surface.json`
  - canary, parallel-wave, rollback, and takeover planning evidence
- `clawhost_workflows.go`
  - workflow contract for skill sync, IM channel validation, device approval, and evidence export

## Branch evidence

- PR: `https://github.com/OpenAGIs/BigClaw/pull/186`
- Branch: `symphony/BIG-PAR-291`
- Sync audit command:
  `bash scripts/ops/bigclawctl github-sync status --json`

## Boundary

- These artifacts define the Go reviewer contract for ClawHost control-plane
  surfaces in this repo.
- They do not execute a live external ClawHost API crawl or perform live bot
  lifecycle mutations from this checkout.
