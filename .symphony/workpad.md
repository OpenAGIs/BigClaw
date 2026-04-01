# BIG-GO-1062 Workpad

## Plan
- Inventory the suggested residual Python assets under `src/bigclaw` and confirm which files are already gone versus still physically present.
- Remove the modules in this batch that no longer participate in a live Python runtime path: `src/bigclaw/planning.py` and `src/bigclaw/operations.py`.
- Remove package exports and Python tests that only exercised the deleted modules, and clean up adjacent references that still point at those deleted Python surfaces.
- Add a Go regression tranche that locks the purge and points at the canonical Go replacement files for planning and operations capabilities.
- Run targeted validation, record exact commands and results, then commit and push the branch.

## Acceptance
- The issue batch inventory is explicit: suggested files are classified as already absent, removed in this change, or retained with rationale.
- `src/bigclaw/planning.py` and `src/bigclaw/operations.py` are deleted, and no package export or Python test still imports them.
- Go regression coverage asserts the deleted Python files stay absent and that the Go replacement surfaces exist.
- `.py` file count decreases relative to the pre-change baseline.

## Validation
- Capture pre/post `.py` file counts with `rg --files . | rg '\\.py$' | wc -l`.
- Run targeted Go tests covering the new purge regression.
- Run a targeted Python test slice that still exercises the surviving neighboring modules after the package export cleanup.
- Record exact commands and pass/fail outcomes in the closeout response.

## Batch Inventory
- Already absent before this change: `src/bigclaw/issue_archive.py`, `src/bigclaw/mapping.py`, `src/bigclaw/memory.py`, `src/bigclaw/orchestration.py`, `src/bigclaw/parallel_refill.py`, `src/bigclaw/pilot.py`, `src/bigclaw/queue.py`, `src/bigclaw/repo_board.py`, `src/bigclaw/repo_commits.py`, `src/bigclaw/repo_gateway.py`, `src/bigclaw/repo_governance.py`
- Removed in this batch: `src/bigclaw/operations.py`, `src/bigclaw/planning.py`
- Retained for later migration because they are still imported by active Python compatibility paths: `src/bigclaw/models.py`, `src/bigclaw/observability.py`

## Archived Closeout

### BIG-GO-1053

- Baseline code migration landed on `main` at `004de016252d6ca168a45dccda48fc9fa69e27f1`.
- Closeout artifacts for the lane are tracked in:
  - `reports/BIG-GO-1053-validation.md`
  - `reports/BIG-GO-1053-closeout.md`
  - `reports/BIG-GO-1053-status.json`
- Validation recorded for `BIG-GO-1053`:
  - `find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l` -> `0`
  - `find . -name '*.py' | wc -l` -> `46`
  - `cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...` -> passed
- Historical branch handoff URL:
  - `https://github.com/OpenAGIs/BigClaw/compare/main...symphony/BIG-GO-1053-validation?expand=1`
- Historical evidence branch `symphony/BIG-GO-1053-validation` has been deleted after
  the closeout landed on `main`.
- Repo-side closeout for `BIG-GO-1053` is complete; the archived notes remain here to avoid losing lane evidence while `main` has moved on to later issues.
