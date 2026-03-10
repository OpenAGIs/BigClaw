# BigClaw

BigClaw is a Symphony/Codex workflow project scaffolded from `workflow.md`.

## What is included

- `workflow.md`: Linear-driven unattended workflow configuration
- `docs/issue-plan.md`: Epic/Issue decomposition from BigClaw PRD v1.0
- `src/bigclaw`: v0.1 foundation modules
  - unified task model
  - persistent priority queue
  - risk/tool based scheduler
  - workflow DSL plus workflow engine with workpad journal and acceptance gate
  - observability ledger with logs/trace/artifact/audit capture
  - queue-to-scheduler execution recording with audit reports
  - benchmark runner with replay, weighted scoring, and version comparison
  - report renderer, issue-close validation gate, and pilot ROI scorecard/portfolio renderer
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
