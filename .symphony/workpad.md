# BIG-GO-1103

## Plan
- confirm the lane file list for the runtime/service/orchestration sweep and verify which candidate Python assets are already gone versus still present
- remove the remaining lane-aligned Python runtime asset that has a clear Go owner and no active source imports
- update active repo guidance so it stops describing the deleted Python runtime module as a live frozen surface
- add a focused Go regression test that locks the deleted Python module absence to its Go runtime replacement
- run targeted validation, record exact commands and outcomes, then commit and push the scoped change set

## Acceptance
- lane coverage is explicit: `src/bigclaw/cost_control.py`, `src/bigclaw/orchestration.py`, and `src/bigclaw/queue.py` were already absent when work started, and this change removes the remaining lane-adjacent residual `src/bigclaw/runtime.py`
- the change deletes a real Python asset instead of doing tracker-only or doc-only cleanup
- repository guidance no longer states that `src/bigclaw/runtime.py` is a retained frozen module
- `find . -name '*.py' | wc -l` decreases from the pre-change baseline
- targeted validation records exact commands and results, with residual risks called out if anything remains

## Validation
- `find . -name '*.py' | wc -l` -> post-change `16`; pre-change baseline was `17` from the initial repository scan
- `rg -n "src/bigclaw/runtime\\.py|src/bigclaw/orchestration\\.py|src/bigclaw/queue\\.py|src/bigclaw/cost_control\\.py" README.md src bigclaw-go/internal docs -g '!docs/go-mainline-cutover-issue-pack.md'` -> matches now limited to regression assertions only: `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go:13` for `runtime.py` and `bigclaw-go/internal/regression/top_level_module_purge_tranche1_test.go:14` for the already-removed `cost_control.py`
- `cd bigclaw-go && go test ./internal/regression` -> `ok  	bigclaw-go/internal/regression	1.338s`
- `cd bigclaw-go && go test ./internal/worker` -> `ok  	bigclaw-go/internal/worker	2.166s`
- `cd bigclaw-go && go test ./internal/planning` -> `ok  	bigclaw-go/internal/planning	1.111s`
- `git status --short` -> modified scope limited to `.symphony/workpad.md`, `README.md`, `bigclaw-go/internal/planning/planning.go`, `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`, `src/bigclaw/planning.py`, and deleted `src/bigclaw/runtime.py`
