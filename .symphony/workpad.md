## BIG-GO-1071

### Plan
- Remove Python packaging residue that still exposes `bigclaw` as a package execution surface.
- Delete `src/bigclaw/__main__.py` so `python -m bigclaw` is no longer a supported default path.
- Delete `src/bigclaw/__init__.py` so the package bootstrap and legacy surface module installation logic are no longer shipped as packaging scaffolding.
- Update Go-side legacy shim and regression tests to reflect the reduced frozen Python surface and point users at `bigclawctl` / `bigclawd`.

### Acceptance
- `src/bigclaw/__main__.py` and `src/bigclaw/__init__.py` are removed.
- No repo code or active test contract still treats `python -m bigclaw` as a maintained entrypoint.
- Go-side validation covers the new reduced legacy Python file set.
- Repository `.py` file count decreases from the pre-change baseline.

### Validation
- `rg -n "python -m bigclaw" README.md bigclaw-go docs src tests scripts .github`
- `rg --files -g '*.py' . | wc -l`
- `cd bigclaw-go && go test ./internal/legacyshim ./internal/regression ./cmd/bigclawctl`

## Archived Closeout

### BIG-GO-1053

- Baseline code migration landed on `main` at `004de016252d6ca168a45dccda48fc9fa69e27f1`.
- Closeout artifacts for the lane are tracked in:
  - `reports/BIG-GO-1053-validation.md`
  - `reports/BIG-GO-1053-closeout.md`
  - `reports/BIG-GO-1053-status.json`
- Additional stale Python entrypoint tests removed after closeout verification:
  - `tests/test_parallel_validation_bundle.py`
  - `tests/test_validation_bundle_continuation_policy_gate.py`
- Validation recorded for `BIG-GO-1053`:
  - `find bigclaw-go/scripts/e2e -maxdepth 1 -name '*.py' | wc -l` -> `0`
  - `find . -name '*.py' | wc -l` -> `43`
  - `cd bigclaw-go && go test ./cmd/bigclawctl/... ./internal/regression/...` -> passed
- Historical branch handoff URL:
  - `https://github.com/OpenAGIs/BigClaw/compare/main...symphony/BIG-GO-1053-validation?expand=1`
- Historical evidence branch `symphony/BIG-GO-1053-validation` has been deleted after
  the closeout landed on `main`.
- Remote closeout comment posted on merged PR `#217`:
  - `https://github.com/OpenAGIs/BigClaw/pull/217#issuecomment-4167169146`
- No writable local tracker entry exists for `BIG-GO-1053` in `local-issues.json` or the
  Symphony local issue store, so any remaining active state is external to this workspace.
- Repo-side closeout for `BIG-GO-1053` is complete; the archived notes remain here to avoid losing lane evidence while `main` has moved on to later issues.
