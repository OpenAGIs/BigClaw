Issue: BIG-GO-1028

Plan
- Retire `tests/test_repo_rollout.py` by moving its remaining coverage into Go-native tests under `bigclaw-go/internal/planning` and `bigclaw-go/internal/reporting`.
- Keep the implementation scoped to the rollout scorecard/candidate-gate helpers and repo narrative export helpers needed by the deleted Python test.
- Delete the migrated Python test file so this tranche reduces repository `.py` inventory immediately.
- Run targeted file-count checks and Go tests; record exact commands and outcomes for final closeout.
- Commit only the scoped issue changes and push the branch to the remote.

Acceptance
- Changes remain scoped to the selected tranche-3 Python test deletion and directly supporting `bigclaw-go/internal/planning` plus `bigclaw-go/internal/reporting` files.
- Repository `.py` file count decreases by deleting the migrated Python test file.
- Repository `.go` file count increases only for the new Go-native rollout coverage files.
- `pyproject.toml`, `setup.py`, and `setup.cfg` remain unchanged.
- Final report includes the impact on `.py` count, `.go` count, and `pyproject/setup*` files.

Validation
- `find . -path './.git' -prune -o -name '*.py' -print | sort | wc -l`
- `find . -path './.git' -prune -o -name '*.go' -print | sort | wc -l`
- `cd bigclaw-go && go test ./internal/planning ./internal/reporting`
- `git diff --stat --cached`
- `git status --short`
