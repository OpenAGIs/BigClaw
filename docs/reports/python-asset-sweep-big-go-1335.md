# BIG-GO-1335 Python Asset Sweep

Date: 2026-04-05

## Inventory

Lane focus:

- `src/bigclaw/*.py`
- `tests/*.py`
- `scripts/*.py`
- `bigclaw-go/scripts/*.py`

Current physical inventory in the active worktree:

- `find . -path './.git' -prune -o -name '*.py' -o -name '*.pyi' -print | wc -l` -> `0`
- `find . -name '*.py' -o -name '*.pyi' | sort` -> no output
- `scripts/` contains shell and Go-first wrappers only
- `bigclaw-go/scripts/` contains shell wrappers plus a Go helper only

## Go Replacement Paths

- Root workspace automation: `bash scripts/ops/bigclawctl ...`
- Root dev smoke: `bash scripts/ops/bigclawctl dev-smoke`
- Refill workflow: `bash scripts/ops/bigclawctl refill --apply --local-issues local-issues.json`
- E2E task smoke: `cd bigclaw-go && go run ./cmd/bigclawctl automation e2e run-task-smoke ...`
- Benchmark automation: `cd bigclaw-go && go run ./cmd/bigclawctl automation benchmark ...`
- Migration automation: `cd bigclaw-go && go run ./cmd/bigclawctl automation migration ...`

## Sweep Changes In This Lane

- Confirmed that repository Python physical asset count is already zero for the targeted directories and the full worktree.
- Removed the default Python command from `bigclaw-go/scripts/e2e/ray_smoke.sh` so the retained shell wrapper does not imply a Python runtime requirement.
- Updated migration/handoff docs to use the current Go-only verification path instead of legacy Python shim assertions.

## Validation

- `find . -name '*.py' -o -name '*.pyi' | sort`
- `find . -path './.git' -prune -o -name '*.py' -o -name '*.pyi' -print | wc -l`
- `rg -n "python3?|\\.py\\b" README.md docs bigclaw-go/scripts scripts`
- `bash bigclaw-go/scripts/e2e/ray_smoke.sh`
