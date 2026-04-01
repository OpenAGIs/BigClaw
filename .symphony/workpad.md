# BIG-GO-1057 Workpad

## Plan
- Confirm every live entry surface that still references `scripts/ops/bigclaw_github_sync.py`.
- Remove `scripts/ops/bigclaw_github_sync.py`.
- Update hooks, README, migration docs, and regression tests to use `bash scripts/ops/bigclawctl github-sync ...` instead of the deleted Python wrapper.
- Add or adjust regression coverage so this slice asserts the deleted wrapper stays absent and the Go-first entrypoint remains usable.
- Run targeted validation, record exact commands and results, then commit and push the branch.

## Acceptance
- `scripts/ops/bigclaw_github_sync.py` is deleted from the repo.
- Live operator entry surfaces no longer call the deleted Python wrapper.
- README, hooks, workflow-adjacent docs, and CI-facing references for this entrypoint point at `scripts/ops/bigclawctl` or equivalent shell/Go entry.
- Regression coverage pins the removal so the Python entrypoint does not return.
- `.py` file count decreases relative to the pre-change baseline.

## Validation
- Capture pre/post `.py` file counts with `rg --files . | rg '\\.py$' | wc -l`.
- Run targeted Go tests covering the github-sync CLI and purge regression.
- Run the Go-first github-sync help/status commands through `scripts/ops/bigclawctl`.
- Record exact commands and pass/fail outcomes in the closeout response.

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
- Repo-side closeout for `BIG-GO-1053` is complete; the archived notes remain here to avoid losing lane evidence while `main` has moved on to later issues.
