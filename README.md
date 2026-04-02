# BigClaw

BigClaw is a Symphony/Codex workflow project scaffolded from `workflow.md`.

`bigclaw-go` is the current implementation mainline for new development. The
repository root now exposes Go-only build entrypoints; legacy Python surfaces
remain migration-only source assets and are not packaged from the root.

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

## Root Go quick start (recommended)

```bash
cd BigClaw
make test
make build
make run
curl localhost:8080/healthz
bash scripts/ops/bigclawctl github-sync status --json
bash scripts/ops/bigclawctl dev-smoke
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
- `bash scripts/ops/bigclawctl refill ...` is the supported refill entrypoint, and the legacy
  `scripts/ops/*workspace*.py` helpers remain compatibility shims over the same Go CLI.
- GitHub sync is no longer exposed through a Python wrapper; use
  `bash scripts/ops/bigclawctl github-sync ...`.
- `go run ./bigclaw-go/cmd/bigclawctl automation e2e run-task-smoke ...`,
  `go run ./bigclaw-go/cmd/bigclawctl automation benchmark soak-local ...`,
  `go run ./bigclaw-go/cmd/bigclawctl automation benchmark run-matrix ...`,
  `go run ./bigclaw-go/cmd/bigclawctl automation benchmark capacity-certification ...`,
  and `go run ./bigclaw-go/cmd/bigclawctl automation migration shadow-compare ...`
  are the supported automation entrypoints. `bigclaw-go/scripts/benchmark/` is
  now Go-only and keeps `run_suite.sh` as the retained wrapper; the migration matrix lives in
  [`bigclaw-go/docs/go-cli-script-migration.md`](./bigclaw-go/docs/go-cli-script-migration.md).
- `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-symphony`, and `scripts/ops/bigclaw-panel` are
  retained as compatibility wrappers, but the preferred operator path is now `scripts/ops/bigclawctl`.
- `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json` promotes the next
  queued local issues to `In Progress` using the canonical order in `docs/parallel-refill-queue.json`.

## Legacy Python migration note

Do not use Python packaging from the repository root. When a migration-only
Python surface must be exercised, validate it directly from source:

```bash
bash scripts/ops/bigclawctl legacy-python compile-check --json
```

Or use the bootstrap helper to validate Go first and then run the legacy
Python migration surface from the active environment without editable install
or repo-root packaging bootstrap:

```bash
BIGCLAW_ENABLE_LEGACY_PYTHON=1 bash scripts/dev_bootstrap.sh
```

That helper always runs the Go `bigclawctl dev-smoke` replacement first, then
`cd bigclaw-go && go test ./internal/bootstrap`, and finally the remaining
legacy Python shim compile-check when `python3` is available.

## Go smoke verify

```bash
cd BigClaw
make test
make run &
curl localhost:8080/healthz
bash scripts/ops/bigclawctl github-sync status --json
```

## Go Dev Smoke Verify

Use this to verify the root dev smoke path:

```bash
bash scripts/ops/bigclawctl dev-smoke
```

## Quality gates

Go mainline:

```bash
make test
make build
```

Go-first bootstrap helper:

```bash
bash scripts/dev_bootstrap.sh
```

Legacy Python migration surface:

```bash
bash scripts/ops/bigclawctl legacy-python compile-check --json
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
The root Go-only build entrypoints are `make test`, `make build`, and `make run`;
the Go-first operator entrypoint is `scripts/ops/bigclawctl`; legacy Python
ops wrappers remain only as compatibility shims during migration, except
GitHub sync which is now Go/shell-only via `scripts/ops/bigclawctl`.

The legacy Python execution-kernel modules in `src/bigclaw/runtime.py`,
`src/bigclaw/scheduler.py`, `src/bigclaw/workflow.py`,
`src/bigclaw/orchestration.py`, and `src/bigclaw/queue.py` are now frozen for
migration-only reference use. Active runtime development belongs in
`bigclaw-go/internal/*`; use `go run ./bigclaw-go/cmd/bigclawd` for the local
server path.
