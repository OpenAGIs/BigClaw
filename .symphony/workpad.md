## BIG-GO-905 Workpad

### Plan

- [x] Audit the current Python `repo_governance`, `repo_board`, and `repo_triage` surfaces alongside the existing Go `internal/triage` and governance primitives to define the migration boundary.
- [x] Implement the first Go-owned repo capability surface for governance, board, and triage in a scoped package set with tests that mirror the Python behavior.
- [x] Write the executable migration plan and cutover checklist covering implementation inventory, validation commands, regression surface, branch/PR guidance, and known risks.
- [x] Run targeted validation, capture exact commands/results, and prepare a scoped commit for branch push.

### Acceptance

- [x] A concrete migration plan exists in-repo for moving repo governance/board/triage from Python to Go, with an initial implementation list.
- [x] Go now owns the first repo governance and repo board capability primitives, and repo triage remains covered in the same capability family.
- [x] Validation commands and the regression surface are explicit and reproducible.
- [x] Branch, PR, and rollout risks are documented for follow-on slices.

### Validation

- [x] `cd bigclaw-go && go test ./internal/repo/...`
  - Result: `ok  	bigclaw-go/internal/repo	1.780s`
- [x] `cd bigclaw-go && go test ./internal/triage ./internal/governance`
  - Result: `ok  	bigclaw-go/internal/triage	2.323s`
  - Result: `ok  	bigclaw-go/internal/governance	2.773s`
- [x] `PYTHONPATH=src python3 -m pytest tests/test_repo_board.py tests/test_repo_governance.py tests/test_repo_triage.py`
  - Result: `5 passed in 0.13s`

### Notes

- Scope is limited to repo governance/board/triage migration planning plus the first Go capability implementation slice needed to unblock follow-on parity work.
- Branch for this issue: `big-go-905`
