# BIG-GO-1080 Workpad

## Plan
- Confirm which residual `tests/*.py` files already have repo-native Go coverage strong enough to replace them without reducing behavioral protection.
- Remove the residual Python tranche that is already covered by Go tests: live-shadow bundle export, queue control-center rendering, orchestration policy/handoff behavior, and file-backed queue persistence.
- Remove the next residual Python tranche that is already covered by Go tests: repo commit-link surfaces, risk scoring, scheduler routing, and worker runtime routing/policy coverage.
- Add small Go-native replacements for the remaining narrow repo test surfaces where direct parity is feasible: collaboration thread merge and pilot rollout / repo narrative helpers.
- Remove the residual `test_models.py` lane now that equivalent Go round-trip coverage already exists in `internal/risk`, `internal/triage`, `internal/workflow`, and `internal/billing`.
- Add a Go regression test that asserts this Python tranche stays deleted so the repo does not silently restore these default Python test entrypoints.
- Run targeted validation for the affected Go packages and record exact commands and results, then verify the repo `.py` count dropped.
- Commit the scoped change set and push the branch to the remote.

## Acceptance
- The residual Python tranche is removed from `tests/`, reducing the repository `.py` count.
- Equivalent or stronger Go-only validation remains in place for the removed tranche.
- A Go regression test fails if the removed Python test files are reintroduced.
- The second removal tranche further reduces residual Python test entrypoints without widening the scope beyond covered Go surfaces.
- The repo-collaboration and repo-rollout Python tests are replaced by new Go-native package tests and removed from `tests/`.
- The Python models round-trip test is removed because the equivalent Go model-contract tests are already present and validated.
- Validation proves the affected Go packages still pass after the Python test removal.

## Validation
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/billing ./internal/collaboration ./internal/pilot ./internal/queue ./internal/reporting ./internal/regression ./internal/repo ./internal/risk ./internal/scheduler ./internal/triage ./internal/worker ./internal/workflow`
- `git status --short`
