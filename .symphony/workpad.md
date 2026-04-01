# BIG-GO-1080 Workpad

## Plan
- Confirm which residual `tests/*.py` files already have repo-native Go coverage strong enough to replace them without reducing behavioral protection.
- Remove the residual Python tranche that is already covered by Go tests: live-shadow bundle export, queue control-center rendering, orchestration policy/handoff behavior, and file-backed queue persistence.
- Remove the next residual Python tranche that is already covered by Go tests: repo commit-link surfaces, risk scoring, scheduler routing, and worker runtime routing/policy coverage.
- Add small Go-native replacements for the remaining narrow repo test surfaces where direct parity is feasible: collaboration thread merge and pilot rollout / repo narrative helpers.
- Remove the residual `test_models.py` lane now that equivalent Go round-trip coverage already exists in `internal/risk`, `internal/triage`, `internal/workflow`, and `internal/billing`.
- Add a small Go-native `internal/evaluation` package for benchmark/replay runner coverage so `test_evaluation.py` can be removed with actual replacement tests.
- Add a Go-native `internal/planning` package for the remaining planning manifest, gate-evaluation, and four-week execution-plan coverage so `tests/test_planning.py` can be removed with direct parity.
- Top up `internal/reporting` with any missing repo-operations metrics coverage needed to remove `tests/test_operations.py` without dropping backend validation.
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
- The Python evaluation lane is replaced by Go-native benchmark/replay tests and removed from `tests/`.
- The Python planning lane is replaced by Go-native planning tests and removed from `tests/`.
- The Python operations lane is removed only if the existing Go reporting coverage plus any narrow top-up tests fully cover the backend/reporting behaviors from `tests/test_operations.py`.
- Validation proves the affected Go packages still pass after the Python test removal.

## Validation
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l`
- `cd bigclaw-go && go test ./internal/planning ./internal/regression`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/billing ./internal/collaboration ./internal/evaluation ./internal/pilot ./internal/planning ./internal/queue ./internal/reporting ./internal/regression ./internal/repo ./internal/risk ./internal/scheduler ./internal/triage ./internal/worker ./internal/workflow`
- `git status --short`

## Validation Results
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l` -> `30`
- `cd bigclaw-go && go test ./internal/planning ./internal/regression` -> `ok   bigclaw-go/internal/planning 0.815s` and `ok   bigclaw-go/internal/regression (cached)`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/billing ./internal/collaboration ./internal/evaluation ./internal/pilot ./internal/planning ./internal/queue ./internal/reporting ./internal/regression ./internal/repo ./internal/risk ./internal/scheduler ./internal/triage ./internal/worker ./internal/workflow` -> all listed packages `ok` with cached reuse where applicable
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l` -> `29`
- `cd bigclaw-go && go test ./internal/reporting ./internal/regression` -> `ok   bigclaw-go/internal/reporting 0.447s` and `ok   bigclaw-go/internal/regression (cached)`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/billing ./internal/collaboration ./internal/evaluation ./internal/pilot ./internal/planning ./internal/queue ./internal/reporting ./internal/regression ./internal/repo ./internal/risk ./internal/scheduler ./internal/triage ./internal/worker ./internal/workflow` -> all listed packages `ok` with cached reuse where applicable
