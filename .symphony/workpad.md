# BIG-GO-1099

## Plan

- remove the final tracked Python file at `src/bigclaw/__init__.py`
- update Go regression coverage that still hard-codes `src/bigclaw/__init__.py`
  as the residual Python surface so the repo-wide floor becomes zero
- validate the legacy shim compile check still short-circuits cleanly with no
  Python files present
- run targeted regression validation and record exact command results
- commit and push the branch

## Acceptance

- `git ls-files '*.py'` returns no tracked Python files
- repo regression coverage fails if any tracked `.py` file is reintroduced
- removal of `src/bigclaw/__init__.py` is covered by targeted Go regression
  tests
- tracked repository `.py` count becomes `0`

## Validation

- `go test ./bigclaw-go/internal/regression ./bigclaw-go/internal/planning ./bigclaw-go/internal/legacyshim`
  - result: failed at repo root (`go: cannot find main module`)
- `cd bigclaw-go && go test ./internal/regression ./internal/planning ./internal/legacyshim`
  - result: `ok   bigclaw-go/internal/regression 1.336s`
  - result: `ok   bigclaw-go/internal/planning 0.794s`
  - result: `ok   bigclaw-go/internal/legacyshim (cached)`
- `find . -name '*.py' | wc -l`
  - result: `0`
- `git add .symphony/workpad.md bigclaw-go/internal/planning/planning.go bigclaw-go/internal/planning/planning_test.go bigclaw-go/internal/regression/python_floor_guard_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche16_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche17_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche18_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche19_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche20_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche21_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche22_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche23_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche24_test.go bigclaw-go/internal/regression/top_level_module_purge_tranche25_test.go src/bigclaw/__init__.py && git ls-files '*.py' | wc -l`
  - result: `0`
- `git commit -m "BIG-GO-1099 remove final python module"`
  - result: commit `4a11b53c3337b25743b519b36f335e936a091eb3`
- `git push origin symphony/BIG-GO-1099`
  - result: pushed `29e67c34..4a11b53c`
