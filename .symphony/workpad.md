# BIG-GO-1080 Workpad

## Plan
- Confirm which residual `tests/*.py` files already have repo-native Go coverage strong enough to replace them without reducing behavioral protection.
- Remove the residual Python tranche that is already covered by Go tests: live-shadow bundle export, queue control-center rendering, orchestration policy/handoff behavior, and file-backed queue persistence.
- Add a Go regression test that asserts this Python tranche stays deleted so the repo does not silently restore these default Python test entrypoints.
- Run targeted validation for the affected Go packages and record exact commands and results, then verify the repo `.py` count dropped.
- Commit the scoped change set and push the branch to the remote.

## Acceptance
- The residual Python tranche is removed from `tests/`, reducing the repository `.py` count.
- Equivalent or stronger Go-only validation remains in place for the removed tranche.
- A Go regression test fails if the removed Python test files are reintroduced.
- Validation proves the affected Go packages still pass after the Python test removal.

## Validation
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/queue ./internal/reporting ./internal/regression ./internal/scheduler ./internal/workflow`
- `git status --short`
