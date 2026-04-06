# BIG-GO-1504 Validation

Date: 2026-04-06

## Scope

Issue: `BIG-GO-1504`

Title: `Refill: reduce scripts and scripts/ops physical Python wrapper count even when wrappers are compatibility-only`

This lane audited the physical Python wrapper inventory under `scripts/` and
`scripts/ops/`, including compatibility-only operator entrypoints.

The checked-out workspace was already at `0` physical Python files repository
wide, and specifically at `0` physical `.py` files in `scripts/` and
`scripts/ops/`. There was therefore no in-branch `.py` wrapper left to delete.
The delivered work records that repository reality and adds a targeted
regression guard plus issue-specific validation evidence.

## Before And After Counts

- Repository-wide physical `.py` files before: `0`
- Repository-wide physical `.py` files after: `0`
- `scripts/*.py` before: `0`
- `scripts/*.py` after: `0`
- `scripts/ops/*.py` before: `0`
- `scripts/ops/*.py` after: `0`
- Deleted physical `.py` files: `none`

## Active Non-Python Replacement Surface

- Regression guard: `bigclaw-go/internal/regression/big_go_1504_script_wrapper_sweep_test.go`
- Lane report: `bigclaw-go/docs/reports/big-go-1504-script-wrapper-sweep.md`
- Root operator entrypoint: `scripts/ops/bigclawctl`
- Root issue helper: `scripts/ops/bigclaw-issue`
- Root panel helper: `scripts/ops/bigclaw-panel`
- Root symphony helper: `scripts/ops/bigclaw-symphony`
- Root bootstrap helper: `scripts/dev_bootstrap.sh`

## Validation Commands

- `rg --files | rg '^(scripts|scripts/ops)/.*\.py$' | wc -l`
- `rg --files | rg '\.py$' | wc -l`
- `find scripts -maxdepth 3 -type f | sort`
- `sed -n '1,120p' scripts/ops/bigclawctl`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1504/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1504(ScriptsAndOpsStayPythonFree|CompatibilityLaunchersRemainNonPython|LaneReportCapturesRepoReality)$'`
- `git push -u origin BIG-GO-1504`

## Validation Results

### Scoped wrapper inventory

Command:

```bash
rg --files | rg '^(scripts|scripts/ops)/.*\.py$' | wc -l
```

Result:

```text
0
```

### Repository-wide Python inventory

Command:

```bash
rg --files | rg '\.py$' | wc -l
```

Result:

```text
0
```

### Live scripts tree

Command:

```bash
find scripts -maxdepth 3 -type f | sort
```

Result:

```text
scripts/dev_bootstrap.sh
scripts/ops/bigclaw-issue
scripts/ops/bigclaw-panel
scripts/ops/bigclaw-symphony
scripts/ops/bigclawctl
```

### Compatibility launcher verification

Command:

```bash
sed -n '1,120p' scripts/ops/bigclawctl
```

Result:

```text
#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
repo_root=$(cd -- "$script_dir/../.." && pwd)
invocation_dir=$(pwd)

cd "$repo_root/bigclaw-go"

needs_repo=true
out_args=()
consume_next_as_repo=0
for arg in "$@"; do
  if [ "$consume_next_as_repo" -eq 1 ]; then
    consume_next_as_repo=0
    if [[ "$arg" != /* && "$arg" != "~"* ]]; then
      if resolved_repo=$(cd "$invocation_dir" && cd "$arg" 2>/dev/null && pwd); then
        out_args+=("$resolved_repo")
      else
        out_args+=("$arg")
      fi
    else
      out_args+=("$arg")
    fi
    continue
  fi
  case "$arg" in
    --repo|--repo=*)
      needs_repo=false
      case "$arg" in
        --repo)
          out_args+=("$arg")
          consume_next_as_repo=1
          ;;
        --repo=*)
          repo_value="${arg#--repo=}"
          if [[ "$repo_value" != /* && "$repo_value" != "~"* ]]; then
            if resolved_repo=$(cd "$invocation_dir" && cd "$repo_value" 2>/dev/null && pwd); then
              out_args+=(--repo="$resolved_repo")
            else
              out_args+=("$arg")
            fi
          else
            out_args+=("$arg")
          fi
          ;;
      esac
      ;;
    *)
      out_args+=("$arg")
      ;;
  esac
done

if [ "$needs_repo" = true ]; then
  exec go run ./cmd/bigclawctl "${out_args[@]}" --repo "$repo_root"
fi

exec go run ./cmd/bigclawctl "${out_args[@]}"
```

### Targeted regression guard

Command:

```bash
cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1504/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1504(ScriptsAndOpsStayPythonFree|CompatibilityLaunchersRemainNonPython|LaneReportCapturesRepoReality)$'
```

Result:

```text
ok  	bigclaw-go/internal/regression	0.185s
```

### Push

Command:

```bash
git push -u origin BIG-GO-1504
```

Result:

```text
branch 'BIG-GO-1504' set up to track 'origin/BIG-GO-1504'.
```

## Git

- Branch: `BIG-GO-1504`
- Lane commit: `264ea883e1f833e08b9f86c1af8e0b53d16ceb17`
- Baseline materialized main commit: `a63c8ec0f999d976a1af890c920a54ac2d6c693a`
- Push target: `origin/BIG-GO-1504`

## Residual Blocker

- The materialized repository baseline was already fully Python-free, including
  `scripts/` and `scripts/ops/`, so `BIG-GO-1504` could not numerically reduce
  the physical `.py` file count in-branch. The lane therefore closes by
  recording exact `0 -> 0` evidence, an explicit empty deleted-file ledger, and
  non-Python ownership of the compatibility launchers.
