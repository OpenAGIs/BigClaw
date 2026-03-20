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
```

## Local orchestration quick start

BigClaw now defaults to a repo-native local tracker in [`local-issues.json`](./local-issues.json).
Use these entrypoints to keep the remaining Go-mainline migration slices moving without waiting on
Linear issue capacity:

```bash
bash scripts/ops/bigclaw-issue list
bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json
bash scripts/ops/bigclaw-symphony
bash scripts/ops/bigclaw-panel
```

Notes:

- `bash scripts/ops/bigclaw-symphony` starts Symphony against [`workflow.md`](./workflow.md) and
  serves the local issue dashboard at `http://127.0.0.1:4000/`.
- `bash scripts/ops/bigclaw-panel` prints the configured dashboard URL for the current workflow.
- `bash scripts/ops/bigclaw-issue ...` wraps `symphony issue ... --workflow workflow.md` so local
  issue creation and state changes stay pinned to this repository's tracker file.
- `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json` promotes the next
  queued local issues to `In Progress` using the canonical order in `docs/parallel-refill-queue.json`.
- `bash scripts/ops/bigclawctl local-issue update --local-issues local-issues.json --issue BIG-GOM-307 --comment-file comment.md`
  records multiline validation evidence without shell-escaping the tracker comment body. Use
  `--comment-file -` to read the comment from stdin.

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

Or use the helper script:

```bash
bash scripts/dev_bootstrap.sh
```

## Legacy Python local test (without editable install)

If your environment has restrictive system-packages permissions, run tests with `PYTHONPATH`:

```bash
PYTHONPATH=src python3 -m pytest
```

## Smoke verify

```bash
PYTHONPATH=src python3 scripts/dev_smoke.py
```

## Quality gates

Go mainline:

```bash
cd BigClaw/bigclaw-go
go test ./...
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
