# BIG-GO-214 Python Asset Sweep

## Scope

`BIG-GO-214` (`Residual scripts Python sweep Q`) removes the last repo-root
alias wrappers that still duplicated `scripts/ops/bigclawctl` for issue,
panel, and symphony entrypoints:

- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`

The repository was already physically Python-free before this lane. The
remaining issue surface was the transitional wrapper/helper layer and the
operator docs that still advertised those aliases as supported.

## Remaining Python Inventory

Repository-wide Python file count: `0`.

- `scripts`: `0` Python files
- `scripts/ops`: `0` Python files
- `bigclaw-go/scripts`: `0` Python files

Retired root wrapper aliases:

- `scripts/ops/bigclaw-issue`
- `scripts/ops/bigclaw-panel`
- `scripts/ops/bigclaw-symphony`

## Canonical Go Or Native Entry Points

The supported root helper surface after this sweep is:

- `scripts/ops/bigclawctl`
- `scripts/dev_bootstrap.sh`
- `bigclaw-go/cmd/bigclawctl/main.go`
- `bigclaw-go/cmd/bigclawctl/migration_commands.go`
- `bigclaw-go/internal/refill/queue_markdown.go`
- `docs/go-cli-script-migration-plan.md`
- `docs/parallel-refill-queue.md`

## Validation Commands And Results

- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
  Result: no output; repository-wide Python file count remained `0`.
- `for path in scripts/ops/bigclaw-issue scripts/ops/bigclaw-panel scripts/ops/bigclaw-symphony; do test ! -e "$path" || echo "present: $path"; done`
  Result: no output; all retired root wrapper aliases remained absent.
- `bash scripts/ops/bigclawctl issue --help`
  Result: exits successfully through the canonical `scripts/ops/bigclawctl`
  helper without relying on alias wrappers; usage banner prints
  `usage: bigclawctl issue [flags] [args...]`.
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO214(RepositoryHasNoPythonFiles|RetiredRootWrapperAliasesRemainAbsent|CanonicalRootEntrypointsRemainAvailable|LaneReportCapturesSweepState)$'`
  Result: `ok  	bigclaw-go/internal/regression	0.158s`
- `cd bigclaw-go && go test -count=1 ./internal/refill`
  Result: `ok  	bigclaw-go/internal/refill	2.643s`

## Residual Risk

- This lane removes the repo-root alias wrappers and active documentation
  references, but historical reports for earlier Python-removal issues still
  mention the old aliases as part of preserved audit evidence.
