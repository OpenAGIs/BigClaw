Issue: BIG-GO-1030

Plan
- Fold the residual Python `risk` compatibility surface into `src/bigclaw/runtime.py`, where the scorer already has its only runtime consumer.
- Update package exports and compatibility shims so `bigclaw.risk` still resolves after the standalone file is removed.
- Merge the dedicated `tests/test_risk.py` coverage into `tests/test_runtime_matrix.py`, then delete `tests/test_risk.py`.
- Refresh directly coupled docs that still point at `src/bigclaw/risk.py` as a standalone Python source asset.
- Re-run targeted runtime/observability pytest slices, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/risk.py` and `tests/test_risk.py` are removed from the tree.
- `bigclaw.risk` imports still resolve through package compatibility shims.
- Runtime risk scoring behavior remains unchanged after the fold.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_runtime_matrix.py tests/test_observability.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nimport bigclaw.risk\nscore = bigclaw.risk.RiskScorer().score_task(bigclaw.Task(task_id=\"t\", source=\"local\", title=\"x\", description=\"\"))\nprint(bigclaw.risk.RiskScore.__name__)\nprint(score.total, score.level.value, score.requires_approval)\nPY`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
