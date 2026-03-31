Issue: BIG-GO-1030

Plan
- Fold the residual Python `repo_plane` compatibility surface into `src/bigclaw/observability.py`, which already owns run-commit evidence handling.
- Update package exports and install a compatibility shim so `bigclaw.repo_plane` still resolves after the standalone file is removed.
- Merge `tests/test_repo_rollout.py` into existing planning/report test modules, then delete the dedicated file.
- Refresh directly coupled docs that still point at `src/bigclaw/repo_plane.py` as a standalone Python source asset.
- Re-run targeted observability/planning/report pytest slices, recalculate repository `.py` / `.go` / `pyproject` / `setup` counts, then commit and push.

Acceptance
- The repository physical `.py` file count decreases again.
- `src/bigclaw/repo_plane.py` and `tests/test_repo_rollout.py` are removed from the tree.
- `bigclaw.repo_plane` imports still resolve through package compatibility shims.
- Repo-space and run-commit evidence structures still work from the migrated owner module.
- Final report includes the exact impact on `.py` count, `.go` count, and `pyproject.toml` / `setup.py` / `setup.cfg` presence.

Validation
- `PYTHONPATH=src python3 -m pytest tests/test_observability.py tests/test_planning.py tests/test_reports.py -q`
- `PYTHONPATH=src python3 - <<'PY'\nimport bigclaw.repo_plane\nprint(bigclaw.repo_plane.RunCommitLink.__name__)\nprint(bigclaw.repo_plane.RepoSpace(space_id=\"s\", project_key=\"BIG\", repo=\"OpenAGIs/BigClaw\").default_channel_for_task(\"BIG-1\"))\nPY`
- `find . -type f \\( -name '*.py' -o -name '*.go' -o -name 'pyproject.toml' -o -name 'setup.py' -o -name 'setup.cfg' \\) | sed 's#^./##' | awk 'BEGIN{py=0;go=0;pp=0;setup=0} /\\.py$/{py++} /\\.go$/{go++} /pyproject\\.toml$/{pp++} /(setup\\.py|setup\\.cfg)$/{setup++} END{printf("py=%d\\ngo=%d\\npyproject=%d\\nsetup=%d\\n",py,go,pp,setup)}'`
- `git diff --stat && git status --short`
