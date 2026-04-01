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
- Add a focused Go-native task-run observability surface under `internal/observability` so `tests/test_observability.py` can be removed with actual ledger/report parity.
- Add a bounded Go-native report-studio / pilot / issue-closure surface and retire `tests/test_reports.py` from default execution so the remaining Python-only report sublanes stop being default test entrypoints.
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
- The Python observability lane is removed only if Go-native task-run ledger, repo-sync audit, collaboration extraction, and detail/report rendering coverage are in place.
- The Python reports lane should only leave default execution if Go-native coverage exists for the isolated report-studio / pilot / issue-closure block and the remaining uncovered Python report sublanes are moved out of default discovery.
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
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l` -> `28`
- `cd bigclaw-go && go test ./internal/observability ./internal/regression` -> `ok   bigclaw-go/internal/observability 0.756s` and `ok   bigclaw-go/internal/regression (cached)`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/billing ./internal/collaboration ./internal/evaluation ./internal/observability ./internal/pilot ./internal/planning ./internal/queue ./internal/reporting ./internal/regression ./internal/repo ./internal/risk ./internal/scheduler ./internal/triage ./internal/worker ./internal/workflow` -> all listed packages `ok` with cached reuse where applicable
- `find tests -maxdepth 1 -name 'test_*.py' | sort` -> `tests/test_console_ia.py`, `tests/test_design_system.py`, `tests/test_ui_review.py`
- `find . -name '*.py' | sed 's#^./##' | sort | wc -l` -> `28`
- `cd bigclaw-go && go test ./internal/reportstudio ./internal/regression` -> `ok   bigclaw-go/internal/reportstudio (cached)` and `ok   bigclaw-go/internal/regression 0.515s`
- `cd bigclaw-go && go test ./cmd/bigclawctl ./internal/billing ./internal/collaboration ./internal/evaluation ./internal/observability ./internal/pilot ./internal/planning ./internal/queue ./internal/reporting ./internal/reportstudio ./internal/regression ./internal/repo ./internal/risk ./internal/scheduler ./internal/triage ./internal/worker ./internal/workflow` -> all listed packages `ok` with cached reuse where applicable
