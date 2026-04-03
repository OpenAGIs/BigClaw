# BIG-GO-1062 Workpad

## Plan
- Inventory the suggested residual Python assets under `src/bigclaw` and confirm which files are already gone versus still physically present.
- Remove the modules in this batch that still exist physically but only carry schema/observability compatibility payloads: `src/bigclaw/models.py` and `src/bigclaw/observability.py`.
- Replace those physical modules with a single downgraded compatibility aggregation shell and clean up package exports so existing imports continue to resolve without the deleted files.
- Extend the existing Go regression tranche so it locks the purge and points at the canonical Go replacement files for the removed Python surfaces.
- Run targeted validation, record exact commands and results, then commit and push the branch.

## Acceptance
- The issue batch inventory is explicit: suggested files are classified as already absent, removed in this change, or retained with rationale.
- `src/bigclaw/models.py` and `src/bigclaw/observability.py` are deleted, while package-level compatibility imports continue to resolve through `src/bigclaw/_compat_schema.py`.
- Go regression coverage asserts the deleted Python files stay absent and that the Go replacement surfaces exist.
- `.py` file count decreases relative to the pre-change baseline.

## Validation
- Capture pre/post `.py` file counts with `rg --files . | rg '\\.py$' | wc -l`.
- Run targeted Go tests covering the new purge regression.
- Run a targeted Python test slice that still exercises the surviving neighboring modules after the package export cleanup.
- Record exact commands and pass/fail outcomes in the closeout response.

## Batch Inventory
- Already absent before this change: `src/bigclaw/issue_archive.py`, `src/bigclaw/mapping.py`, `src/bigclaw/memory.py`, `src/bigclaw/orchestration.py`, `src/bigclaw/parallel_refill.py`, `src/bigclaw/pilot.py`, `src/bigclaw/queue.py`, `src/bigclaw/repo_board.py`, `src/bigclaw/repo_commits.py`, `src/bigclaw/repo_gateway.py`, `src/bigclaw/repo_governance.py`
- Already absent before this change: `src/bigclaw/operations.py`, `src/bigclaw/planning.py`
- Removed in this batch: `src/bigclaw/models.py`, `src/bigclaw/observability.py`
- Replaced by downgraded compatibility aggregation shell: `src/bigclaw/_compat_schema.py`

## Validation Record
- `git ls-tree -r --name-only HEAD | rg '\\.py$' | wc -l` -> `39`
- `rg --files . | rg '\\.py$' | wc -l` -> `38`
- `python3 -m pytest tests/test_models.py tests/test_observability.py tests/test_reports.py tests/test_runtime_matrix.py` -> `48 passed in 0.09s`
- `python3 -m pytest tests/test_risk.py tests/test_evaluation.py` -> `10 passed in 0.06s`
- `cd bigclaw-go && go test ./internal/regression -run TestTopLevelModulePurgeTranche14 -count=1` -> `ok   bigclaw-go/internal/regression 0.897s`

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
