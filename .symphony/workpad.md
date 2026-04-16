## BIG-GO-1611

### Plan
- Audit the surviving regression coverage around the retired `tests/` Python tree, especially `tests/conftest.py` references and any lane/refill blocker-chain files that still encode the old Python test inventory.
- Replace or consolidate stale guards with a single issue-scoped regression contract that proves the repository remains free of the retired Python test assets and that the Go replacements remain present.
- Delete redundant regression residue only where the replacement contract fully covers the same behavior.
- Run targeted regression tests for the touched area and record the exact commands and outcomes.
- Commit and push the issue branch after validation passes.

### Acceptance
- The checkout contains no surviving `tests/*.py` assets or `tests/conftest.py` dependency chain in the touched regression coverage.
- Any deleted or renamed regression coverage remains fully replaced by repo-native Go assertions.
- No new scope is introduced outside the Python-test/conftest cleanup for this issue.
- Targeted Go tests covering the changed regression contract pass.
- Changes are committed and pushed to the remote branch.

### Validation
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `find tests bigclaw-go/internal/refill bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`
- `cd bigclaw-go && go test -count=1 ./internal/refill`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestPythonTestTranche17Removed|TestBIGGO253|TestBIGGO1611'`
- `git status --short`
- `git log -1 --stat`

### Results
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  - exit `0`; output empty
- `find tests bigclaw-go/internal/refill bigclaw-go/internal/regression -type f -name '*.py' 2>/dev/null | sort`
  - exit `0`; output empty
- `cd bigclaw-go && go test -count=1 ./internal/refill`
  - exit `0`; `ok  	bigclaw-go/internal/refill	2.616s`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestPythonTestTranche17Removed|TestBIGGO253|TestBIGGO1611'`
  - exit `0`; `ok  	bigclaw-go/internal/regression	0.151s`
