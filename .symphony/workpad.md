# BIG-GO-214 Workpad

## Plan

1. Remove the residual root wrapper aliases `scripts/ops/bigclaw-issue`,
   `scripts/ops/bigclaw-panel`, and `scripts/ops/bigclaw-symphony` so
   `scripts/ops/bigclawctl` is the single supported repo-root operator
   entrypoint.
2. Update the active operator docs and generated refill-queue markdown source so
   they stop advertising the removed wrapper aliases and point directly at
   `bash scripts/ops/bigclawctl ...`.
3. Add `BIG-GO-214` issue-scoped regression/report artifacts that record the
   wrapper removal, the retained Go-native helper surfaces, and the exact
   validation commands/results for this lane.

## Acceptance

- `scripts/ops/bigclaw-issue`, `scripts/ops/bigclaw-panel`, and
  `scripts/ops/bigclaw-symphony` are absent from the repo.
- `README.md`, `docs/go-cli-script-migration-plan.md`,
  `bigclaw-go/docs/go-cli-script-migration.md`, and the refill queue markdown
  surfaces no longer present the removed wrapper aliases as supported
  entrypoints.
- `BIG-GO-214` adds regression/report evidence that the repository remains
  Python-free while `scripts/ops/bigclawctl` remains the canonical root helper.

## Validation

- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-214 -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `for path in /Users/openagi/code/bigclaw-workspaces/BIG-GO-214/scripts/ops/bigclaw-issue /Users/openagi/code/bigclaw-workspaces/BIG-GO-214/scripts/ops/bigclaw-panel /Users/openagi/code/bigclaw-workspaces/BIG-GO-214/scripts/ops/bigclaw-symphony; do test ! -e "$path" || echo "present: $path"; done`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-214 && bash scripts/ops/bigclawctl issue --help`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-214/bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO214(RepositoryHasNoPythonFiles|RetiredRootWrapperAliasesRemainAbsent|CanonicalRootEntrypointsRemainAvailable|LaneReportCapturesSweepState)$'`

## Execution Notes

- 2026-04-11: The branch baseline is already physically Python-free; the
  residual issue surface is the remaining root shell alias wrappers and the
  docs/generator text that still advertise them.
