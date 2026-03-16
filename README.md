# BigClaw

BigClaw is a Symphony/Codex workflow project scaffolded from `workflow.md`.

## What is included

- `workflow.md`: Linear-driven unattended workflow configuration
- `docs/symphony-repo-bootstrap-template.md`: repo-agnostic shared mirror + worktree bootstrap template
- `docs/issue-plan.md`: Epic/Issue decomposition from BigClaw PRD v1.0
- `src/bigclaw`: v0.1 foundation modules
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

## Developer quick start (recommended)

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

## Local test (without editable install)

If your environment has restrictive system-packages permissions, run tests with `PYTHONPATH`:

```bash
PYTHONPATH=src python3 -m pytest
```

## Smoke verify

```bash
PYTHONPATH=src python3 scripts/dev_smoke.py
```

## Quality gates

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
The generic entrypoint is `scripts/ops/symphony_workspace_bootstrap.py`; BigClaw keeps
`bigclaw_workspace_bootstrap.py` only as a compatibility wrapper.

