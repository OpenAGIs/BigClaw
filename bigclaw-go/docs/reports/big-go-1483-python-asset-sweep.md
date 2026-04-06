# BIG-GO-1483 Python Asset Sweep

`BIG-GO-1483` records the remaining checked-in caller cutover state for
`bigclaw-go/scripts` after the earlier Python-file removals had already landed.

## Baseline

- Before update physical Python files under `bigclaw-go/scripts`: `0`.
- After update physical Python files under `bigclaw-go/scripts`: `0`.
- Before update checked-in caller references to retired `bigclaw-go/scripts` Python paths: `1`.
- After update checked-in caller references to retired `bigclaw-go/scripts` Python paths: `0`.

## Active `bigclaw-go/scripts` Surface

- `benchmark/run_suite.sh`
- `e2e/run_all.sh`
- `e2e/kubernetes_smoke.sh`
- `e2e/ray_smoke.sh`
- `e2e/broker_bootstrap_summary.go`

All remaining checked-in entrypoints under `bigclaw-go/scripts/` are shell or
Go surfaces that dispatch through `bigclawctl automation ...` or provide Go
report helpers.

## Evidence Commands

- `find bigclaw-go/scripts -type f -name '*.py' | sort`
- `find bigclaw-go/scripts -type f -name '*.py' | wc -l`
- `rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!local-issues.json' --glob '!bigclaw-go/internal/regression/**' --glob '!.symphony/**' 'bigclaw-go/scripts/.*\\.py' README.md docs scripts .github bigclaw-go | sort`
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160|TestBIGGO1483|TestE2E'`

## Blocker

The repository baseline for this branch was already Python-free, so this lane
cannot reduce the physical Python file count below zero. The scoped work here
is limited to removing remaining checked-in caller references and capturing the
exact zero-Python evidence.
