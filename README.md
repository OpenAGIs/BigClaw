# BigClaw

BigClaw is a Symphony/Codex workflow project scaffolded from `workflow.md`.

`bigclaw-go` is the current implementation mainline for new development. The
root Python package is retained only for staged migration and legacy surfaces
that have not been cut over yet.

## What is included

- `workflow.md`: Linear-driven unattended workflow configuration
- `bigclaw-go`: current Go implementation mainline
  - `cmd/bigclawd`: service entrypoint
  - `internal/*`: queue, scheduler, worker, events, API, reporting, and control-plane packages
  - `docs/*`: Go control-plane validation and migration evidence
- `docs/symphony-repo-bootstrap-template.md`: repo-agnostic shared mirror + worktree bootstrap template
- `docs/issue-plan.md`: Epic/Issue decomposition from BigClaw PRD v1.0
- `src/bigclaw`: legacy Python foundation modules pending staged migration to Go
  - engineering operations analytics for dashboards, triage, regressions, and weekly reports
  - `BIG-1606` Policy/Prompt Version Center with workflow/prompt/policy history, diffs, rollback targets, and bundle rendering
  - unified task model
  - persistent priority queue
  - risk/tool based scheduler
  - worker runtime with sandbox profiles and auditable tool gateway
  - workflow DSL plus workflow engine with workpad journal, orchestration artifacts/canvas, entitlement-aware policy, and acceptance gate
  - observability ledger with logs/trace/artifact/audit capture
  - queue-to-scheduler execution recording with audit reports
  - auto triage center for failed, pending-approval, and replay-needed runs, with inbox suggestions, similarity evidence, and reviewer feedback tracking
  - benchmark runner with replay, weighted scoring, and version comparison
  - report renderer, issue-close validation gate, pilot ROI scorecard/portfolio renderer, human takeover queue reporting, ledger-driven orchestration portfolio rollups, and HTML overview pages
  - narrative report studio with section composing plus markdown, HTML, and plain-text export
  - v2 design-system token/component inventory with release-readiness audit reporting
- `tests/`: unit tests

## Go mainline quick start (recommended)

```bash
cd BigClaw/bigclaw-go
go test ./...
go run ./cmd/bigclawd
curl localhost:8080/healthz
bash ../scripts/ops/bigclawctl github-sync status --json
bash ../scripts/ops/bigclawctl dev-smoke
```

## Local orchestration quick start

BigClaw now defaults to a repo-native local tracker in [`local-issues.json`](./local-issues.json).
Use these entrypoints to keep the remaining Go-mainline migration slices moving without waiting on
Linear issue capacity:

```bash
bash scripts/ops/bigclawctl issue list
bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json
bash scripts/ops/bigclawctl symphony
bash scripts/ops/bigclawctl panel
```

Notes:

- `bash scripts/ops/bigclawctl symphony` starts Symphony against [`workflow.md`](./workflow.md) and
  serves the local issue dashboard at `http://127.0.0.1:4000/`.
- `bash scripts/ops/bigclawctl panel` prints the configured dashboard URL for the current workflow.
- `bash scripts/ops/bigclawctl issue ...` wraps `symphony issue ... --workflow workflow.md` so local
  issue creation and state changes stay pinned to this repository's tracker file.
- `python3 scripts/create_issues.py` and `python3 scripts/dev_smoke.py` are now
  compatibility shims that dispatch into `bigclawctl` Go subcommands.
- `python3 scripts/ops/bigclaw_github_sync.py ...`,
  `python3 scripts/ops/bigclaw_refill_queue.py ...`, and the legacy
  `scripts/ops/*workspace*.py` helpers are also compatibility shims over the same Go CLI.
- `python3 bigclaw-go/scripts/e2e/run_task_smoke.py`,
  `python3 bigclaw-go/scripts/e2e/export_validation_bundle.py`,
  `python3 bigclaw-go/scripts/e2e/validation_bundle_continuation_scorecard.py`,
  `python3 bigclaw-go/scripts/e2e/validation_bundle_continuation_policy_gate.py`,
  `python3 bigclaw-go/scripts/e2e/mixed_workload_matrix.py`,
  `python3 bigclaw-go/scripts/e2e/external_store_validation.py`,
  `python3 bigclaw-go/scripts/e2e/broker_failover_stub_matrix.py`,
  `python3 bigclaw-go/scripts/e2e/subscriber_takeover_fault_matrix.py`,
  `python3 bigclaw-go/scripts/e2e/cross_process_coordination_surface.py`,
  `python3 bigclaw-go/scripts/benchmark/run_matrix.py`,
  `python3 bigclaw-go/scripts/benchmark/capacity_certification.py`,
  `python3 bigclaw-go/scripts/benchmark/soak_local.py`, and
  `python3 bigclaw-go/scripts/migration/shadow_compare.py`,
  `python3 bigclaw-go/scripts/migration/shadow_matrix.py`,
  `python3 bigclaw-go/scripts/migration/live_shadow_scorecard.py`, and
  `python3 bigclaw-go/scripts/migration/export_live_shadow_bundle.py` now forward into
  `bigclawctl automation ...`; the migration matrix lives in
  [`bigclaw-go/docs/go-cli-script-migration.md`](./bigclaw-go/docs/go-cli-script-migration.md).
- `cd bigclaw-go && go run ./cmd/bigclawctl legacy-python inventory --json` prints the
  current script migration inventory, planned waves, compatibility policy, and branch / PR guidance.
- `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-symphony`, and `scripts/ops/bigclaw-panel` are
  retained as compatibility wrappers, but the preferred operator path is now `scripts/ops/bigclawctl`.
- `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json` promotes the next
  queued local issues to `In Progress` using the canonical order in `docs/parallel-refill-queue.json`.

## Legacy Python quick start (migration-only)

> Do not use this path for new mainline development. Use it only when migrating
> a required legacy surface to Go or validating an existing Python-only path.

> Do not use system Python directly for editable install. Use a virtualenv.

```bash
cd BigClaw
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -U pip
pip install -e .[dev]
python -m pytest
```

Or use the legacy bootstrap helper:

```bash
BIGCLAW_ENABLE_LEGACY_PYTHON=1 bash scripts/dev_bootstrap.sh
```

## Legacy Python local test (without editable install)

If your environment has restrictive system-packages permissions, run tests with `PYTHONPATH`:

```bash
PYTHONPATH=src python3 -m pytest
```

## Go smoke verify

```bash
cd BigClaw/bigclaw-go
go test ./...
go run ./cmd/bigclawd &
curl localhost:8080/healthz
bash ../scripts/ops/bigclawctl github-sync status --json
```

## Legacy Python smoke verify

Use this only when validating a frozen migration-reference path:

```bash
bash scripts/ops/bigclawctl dev-smoke
python3 scripts/dev_smoke.py
```

## Quality gates

Go mainline:

```bash
cd BigClaw/bigclaw-go
go test ./...
```

Go-first bootstrap helper:

```bash
bash scripts/dev_bootstrap.sh
```

Legacy Python migration surface:

```bash
ruff check src tests scripts
python -m pytest
python -m build
pre-commit run --all-files
```

## Quick verify

```bash
git log -1 --stat
git remote -v
git push origin main
```

Repository: https://github.com/OpenAGIs/BigClaw

## Repo-agnostic bootstrap template

Use `docs/symphony-repo-bootstrap-template.md` when you want another Symphony-managed repo to
reuse the same local mirror + `git worktree` pattern without inheriting BigClaw-specific names.
The Go-first BigClaw entrypoint is `scripts/ops/bigclawctl`; legacy Python
bootstrap wrappers remain only as compatibility shims during migration.

The legacy Python execution-kernel modules in `src/bigclaw/runtime.py`,
`src/bigclaw/scheduler.py`, `src/bigclaw/workflow.py`,
`src/bigclaw/orchestration.py`, and `src/bigclaw/queue.py` are now frozen for
migration-only reference use. The legacy `python -m bigclaw serve` /
`src/bigclaw/service.py` path is also frozen; use `go run ./bigclaw-go/cmd/bigclawd`
for the active local server path. Active runtime development belongs in
`bigclaw-go/internal/*`.
