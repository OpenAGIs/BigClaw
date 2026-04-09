## Codex Workpad

### Issue

- `BIG-GO-180` — Convergence sweep toward practical Go-only repo state

### Plan

- [x] Audit the remaining active root-level Python scripts, `.py`-named operator wrappers, and Python packaging files that still sit on the documented Go-first path.
- [x] Remove or replace only the active compatibility wrappers that are no longer needed now that `bigclaw-go` is the implementation mainline.
- [x] Update repository docs and Python tooling config so they match the reduced residual Python footprint without touching frozen migration-reference modules under `src/bigclaw` or the Python validation corpus under `bigclaw-go/scripts`.
- [x] Run targeted validation for the updated script entrypoints and packaging metadata, then record exact commands and results here.

### Acceptance Criteria

- [x] Active operator helpers under `scripts/` and `scripts/ops/` no longer require Python script entrypoints or `.py`-named wrappers where a non-Python entrypoint is available.
- [x] Obsolete Python packaging/script leftovers removed by this sweep are no longer referenced by docs or lint/build config.
- [x] The remaining Python surface is explicitly migration-only or test/reference-only, with no regression in the Go-first operator path.
- [x] Targeted validation passes for the touched entrypoints and config surfaces.

### Validation

- [x] `bash scripts/dev-smoke`
  Result: passed; emitted the expected migration-only deprecation warning for `scripts/dev-smoke` and finished with `smoke_ok docker`.
- [x] `bash scripts/ops/bigclaw-github-sync status --json`
  Result: passed; wrapper executed and returned JSON status for the current repo state (`status: "ok"`).
- [x] `bash scripts/ops/bigclaw-refill-queue --local-issues local-issues.json`
  Result: passed; dry-run completed with `queue_drained: true` and no runnable refill candidates.
- [x] `python3 -m pip install -e '.[dev]' --dry-run`
  Result: passed; editable metadata resolved from `pyproject.toml` without `setup.py` and concluded with `Would install bigclaw-0.1.0`.

### Notes

- Scope this issue to residual script/wrapper and packaging cleanup on the active operator path, not to the frozen legacy modules under `src/bigclaw` or the Python test/reference assets under `bigclaw-go/scripts`.
- `find scripts -type f -name '*.py'` now returns no results; the remaining active script entrypoints under `scripts/` and `scripts/ops/` are shell-based.
