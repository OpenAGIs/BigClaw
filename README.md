# BigClaw

BigClaw is a Symphony/Codex workflow project scaffolded from `workflow.md`.

## What is included

- `workflow.md`: Linear-driven unattended workflow configuration
- `docs/issue-plan.md`: Epic/Issue decomposition from BigClaw PRD v1.0
- `src/bigclaw`: v0.1 foundation modules
  - unified task model
  - persistent priority queue
  - risk/tool based scheduler
  - worker runtime with sandbox profiles and auditable tool gateway
  - workflow DSL plus workflow engine with workpad journal and acceptance gate
  - observability ledger with logs/trace/artifact/audit capture
  - queue-to-scheduler execution recording with audit reports
  - benchmark runner with replay, weighted scoring, and version comparison
  - report renderer, issue-close validation gate, and pilot ROI scorecard/portfolio renderer
- `tests/`: unit tests

## Weekly operations report

Use `bigclaw.generate_weekly_operations_report(...)` to aggregate observability ledger runs into a weekly engineering operations summary with throughput, approval backlog, execution mix, and follow-up items.

CLI example:

```bash
python3 scripts/generate_weekly_ops_report.py \
  --ledger reports/observability.json \
  --out reports/weekly-ops.md \
  --team OpenAGI \
  --period-start 2026-03-03T00:00:00Z \
  --period-end 2026-03-09T23:59:59Z
```

## Local test

```bash
python3 -m pip install -e . pytest
python3 -m pytest
```

## Quick verify

```bash
git log -1 --stat
git remote -v
git push origin main
```

Repository: https://github.com/OpenAGIs/BigClaw
