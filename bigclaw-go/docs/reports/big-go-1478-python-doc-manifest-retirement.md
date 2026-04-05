# BIG-GO-1478 Python doc/template retirement

`BIG-GO-1478` audited the residual docs/examples/manifests surface after the
repo had already reached a zero-`.py` baseline.

## Baseline

- Repository-wide tracked Python file count at branch start: `0`
- Repository-wide physical Python file count at branch start: `0`
- The remaining stale surface was documentation-only:
  `docs/symphony-repo-bootstrap-template.md` still told downstream repos to
  copy `workspace_bootstrap.py` and `workspace_bootstrap_cli.py`.

## Applied change

- Retired the Python bootstrap compatibility guidance from
  `docs/symphony-repo-bootstrap-template.md`.
- Pinned Go ownership for bootstrap execution to:
  - `scripts/ops/bigclawctl`
  - `bigclaw-go/internal/bootstrap/*`
  - `workflow.md`
- Updated migration/cutover docs so they describe the template retirement and
  do not imply those Python bootstrap assets are still expected to exist.

## Delete conditions and ownership

- `workspace_bootstrap.py`: explicit delete condition is unchanged deletion;
  do not recreate it in repo scaffolds that adopt the shared template.
- `workspace_bootstrap_cli.py`: explicit delete condition is unchanged deletion;
  do not recreate it in repo scaffolds that adopt the shared template.
- Replacement ownership remains Go-first under `scripts/ops/bigclawctl` and
  `bigclaw-go/internal/bootstrap/*`.

## Validation

- `git ls-files '*.py'`
  - Result: no output
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
  - Result: no output
- `rg -n 'workspace_bootstrap\\.py|workspace_bootstrap_cli\\.py' docs`
  - Result: historical references remain only in migration-history docs; the
    shared bootstrap template no longer prescribes those files as live assets
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1478|TestBIGGO1299'`
  - Result: pass

## Outcome

This branch could not lower the Python file count below `0`, but it did retire
the remaining active template guidance that would have recreated executable
Python bootstrap assets in downstream repos. That moves the effective repo
reality closer to a durable Go-only posture instead of a reversible zero-file
snapshot.
