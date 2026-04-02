# BIG-GO-1100

## Plan

1. Remove the remaining dormant legacy Python repo-materialization modules under `src/bigclaw/` that are already documented as superseded by Go-owned surfaces, while leaving the explicit compatibility shim path (`legacy_shim.py`) untouched for this issue.
2. Add a Go regression test tranche that proves the deleted Python files stay absent and that the documented Go replacement packages remain present.
3. Refresh the minimal repo documentation that still describes the deleted Python execution-kernel files as frozen references instead of removed assets.
4. Run targeted validation covering the new regression tranche and repository Python-file count evidence, then commit and push the branch.

## Acceptance

- Physical Python asset count under `src/bigclaw/` drops from 17 to 1.
- The removed Python files are represented by repo-native Go replacement evidence in `bigclaw-go/internal/*`.
- A Go regression test fails if any removed Python file returns to the repo.
- README copy no longer claims the deleted runtime files remain as frozen Python references.

## Validation

- `rg --files -g '*.py' src/bigclaw`
- `cd bigclaw-go && go test ./internal/regression`
- `git status --short`

## Results

- `rg --files -g '*.py' src/bigclaw` -> `src/bigclaw/legacy_shim.py`
- `cd bigclaw-go && go test ./internal/regression` -> `ok  	bigclaw-go/internal/regression	0.600s`
- `git status --short` -> modified `.symphony/workpad.md`, `README.md`; deleted 16 `src/bigclaw/*.py` files; added `bigclaw-go/internal/regression/top_level_module_purge_tranche14_test.go`
- Follow-up planner cleanup: rewired `bigclaw-go/internal/planning` candidate validation commands and evidence links away from deleted Python modules/tests to the current Go-native owners.
- `cd bigclaw-go && go test ./internal/planning ./internal/regression` -> `ok  	bigclaw-go/internal/planning	0.406s`; `ok  	bigclaw-go/internal/regression	0.901s`
- `find . -name '*.py' | sort | wc -l` -> `1`
- `rg -n "src/bigclaw/(audit_events|collaboration|console_ia|deprecation|design_system|evaluation|governance|models|observability|operations|planning|reports|risk|run_detail|runtime|ui_review)\\.py|tests/test_(design_system|console_ia|operations|reports|ui_review)\\.py|python3 -m pytest tests/test_(design_system|operations|reports)" bigclaw-go/internal/planning docs/go-mainline-cutover-handoff.md README.md -g '*.go' -g '*.md'` -> only intentional historical/assertion matches remain
