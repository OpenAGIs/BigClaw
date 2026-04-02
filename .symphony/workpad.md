# BIG-GO-1094

## Plan
- inspect the remaining Python planning/test references and the existing Go-native planning replacement
- remove the legacy Python planning module that still carries deleted `tests/*.py` validation metadata
- update Python package exports plus Go planning backlog/tests so release and rollout evidence is Go-only for this tranche
- add or adjust regression coverage proving the removed Python planning surface stays absent and the Go replacements remain wired
- run targeted validation for planning/regression packages and record exact commands plus results
- commit the scoped change set and push the issue branch to remote

## Acceptance
- the repository `.py` file count decreases from the pre-change baseline
- `src/bigclaw/planning.py` is removed and its Python export surface is retired cleanly
- `bigclaw-go/internal/planning` no longer validates against deleted `tests/*.py` targets for this tranche
- Go regression coverage proves the removed Python planning surface stays absent and Go replacement files remain present
- targeted validation passes and exact commands plus results are recorded

## Validation
- `find . -name '*.py' | wc -l`
- `cd bigclaw-go && go test ./internal/planning ./internal/regression`
- `rg -n "tests/test_design_system.py|tests/test_console_ia.py|tests/test_operations.py|tests/test_reports.py|tests/test_ui_review.py" bigclaw-go/internal/planning src/bigclaw`

## Validation Results
- `find . -name '*.py' | wc -l` -> `21` after deletion, down from the pre-change baseline `22`
- `cd bigclaw-go && go test ./internal/planning ./internal/regression` -> `ok   bigclaw-go/internal/planning 0.642s`; `ok   bigclaw-go/internal/regression 0.993s`
- `rg -n "tests/test_design_system.py|tests/test_console_ia.py|tests/test_operations.py|tests/test_reports.py|tests/test_ui_review.py" bigclaw-go/internal/planning src/bigclaw` -> exit `1` with no matches
