Issue: BIG-GO-1028

Plan
- Retire `tests/test_evaluation.py` by adding an isolated Go-native evaluation compatibility surface in a new `bigclaw-go/internal/evaluation` package without modifying the unrelated in-flight edits already present in `internal/reporting` and `internal/planning`.
- Port the benchmark runner, replay, comparison, and rendering assertions into a new Go `_test.go` file that exercises scoring, mismatch reporting, suite deltas, filesystem outputs, and detail/index page rendering.
- Delete the migrated Python test file so this tranche reduces repository `.py` inventory immediately.
- Run targeted file-count checks and Go tests; record exact commands and outcomes for final closeout.
- Commit only the scoped issue changes and push the branch to the remote.

Acceptance
- Changes remain scoped to the selected tranche-3 Python test deletion and directly supporting Go-native evaluation files.
- Repository `.py` file count decreases by deleting the migrated Python test file.
- Repository `.go` file count increases only for the new Go-native evaluation compatibility files.
- `pyproject.toml`, `setup.py`, and `setup.cfg` remain unchanged.
- Final report includes the impact on `.py` count, `.go` count, and `pyproject/setup*` files.

Validation
- `find tests -maxdepth 1 -name 'test_*.py' | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l`
- `cd bigclaw-go && go test ./internal/evaluation -run 'TestBenchmarkRunnerScoresAndReplaysCase|TestBenchmarkRunnerDetectsFailedExpectation|TestReplayOutcomeReportsMismatch|TestSuiteComparisonAndReport|TestRenderReplayDetailPageListsMismatches|TestRenderRunReplayIndexPageLinksOutputs|TestRenderRunReplayIndexPageWithoutReportPath'`
- `git diff --stat`
- `git status --short`
