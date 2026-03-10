# BigClaw

BigClaw is a Symphony/Codex workflow project scaffolded from `workflow.md`.

## What is included

- `workflow.md`: Linear-driven unattended workflow configuration
- `docs/issue-plan.md`: Epic/Issue decomposition from BigClaw PRD v1.0
- `src/bigclaw`: v0.1 foundation modules
  - engineering operations analytics for dashboards, triage, regressions, and weekly reports
  - unified task model
  - persistent priority queue
  - risk/tool based scheduler
  - worker runtime with sandbox profiles and auditable tool gateway
  - workflow DSL plus workflow engine with workpad journal, orchestration artifacts/canvas, entitlement-aware policy, and acceptance gate
  - observability ledger with logs/trace/artifact/audit capture
  - queue-to-scheduler execution recording with audit reports
  - auto triage center for failed, pending-approval, and replay-needed runs
  - benchmark runner with replay, weighted scoring, and version comparison
  - report renderer, issue-close validation gate, pilot ROI scorecard/portfolio renderer, and human takeover queue reporting
- `tests/`: unit tests

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
