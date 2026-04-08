## Codex Workpad

### Issue

- `BIG-GO-184` — Residual scripts Python sweep N

### Plan

- [x] Audit the remaining root-level Python scripts, `.py`-named wrappers, and packaging helpers that are still on the active operator path after the Go cutover.
- [x] Replace active script entrypoints with non-Python equivalents and remove obsolete Python-only compatibility files that are no longer referenced.
- [x] Update repository documentation and Python packaging/lint configuration so the remaining migration-only Python surface matches the reduced script footprint.
- [x] Run targeted validation for the replacement entrypoints and record exact commands/results here.

### Acceptance Criteria

- [x] The active operator helpers in `scripts/` and `scripts/ops/` no longer depend on Python script entrypoints or `.py`-named compatibility wrappers.
- [x] Obsolete Python packaging/script leftovers removed by this sweep are no longer referenced by repo docs or config.
- [x] Targeted validation passes for the updated script entrypoints and any touched Python/Go config surfaces.

### Validation

- [x] `bash scripts/dev-smoke`
  Result: passed; `go test ./...` completed successfully across `bigclaw-go`, including `internal/githubsync`, `internal/refill`, and the full queue/runtime packages.
- [x] `bash scripts/ops/bigclaw-github-sync status --json`
  Result: passed; returned branch `BIG-GO-184` with `status: "ok"`.
- [x] `bash scripts/ops/bigclaw-refill-queue --local-issues local-issues.json`
  Result: passed; dry-run completed with no runnable refill candidates and reported the queue as drained.
- [x] `python3 -m pip install -e '.[dev]' --dry-run`
  Result: passed; editable install metadata resolved from `pyproject.toml` without `setup.py` and concluded with `Would install bigclaw-0.1.0`.

### Notes

- 2026-04-09: Scoped this issue to residual script/wrapper cleanup in the repo root and `scripts/ops`, not to the frozen migration-reference Python modules under `src/bigclaw` or the Python validation corpus under `bigclaw-go/scripts`.
- 2026-04-09: Removed obsolete `scripts/create_issues.py` and `setup.py`, replaced the remaining operator-facing `.py` wrapper names with shell entrypoints, and updated docs/config to point at the non-Python paths.
