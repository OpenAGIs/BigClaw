# BIG-GO-1605

## Plan
1. Confirm which reporting and observability Python-owned surfaces are already absent and which Go replacements now own them.
2. Finish the Go weekly-reporting CLI surface in `bigclaw-go/cmd/bigclawctl` and add focused tests around input loading, rendering, and emitted metadata.
3. Publish issue-scoped refill evidence for the retired reporting/observability Python paths and update the active migration docs to point at the Go API/CLI replacements.
4. Run targeted sweeps and Go tests, capture exact commands/results in the issue report, then commit and push the branch.

## Acceptance
- `bigclawctl reporting weekly` is available from the root CLI and can render weekly operations artifacts from task/event JSON using Go-owned reporting code.
- Reporting and observability migration docs describe the current Go surfaces, including the CLI/API paths that replace the retired Python-owned weekly reporting flow.
- BIG-GO-1605 adds issue-scoped regression coverage that guards the retired reporting/observability Python assets and the replacement Go paths.
- Validation artifacts record the exact repository sweep and targeted test commands with their observed results.
- The branch is committed and pushed to `origin/BIG-GO-1605`.

## Validation
- `find /Users/openagi/code/bigclaw-workspaces/BIG-GO-1605 -path '*/.git' -prune -o -name '*.py' -type f -print | sort`
- `for path in src/bigclaw/observability.py src/bigclaw/reports.py src/bigclaw/evaluation.py src/bigclaw/operations.py tests/test_observability.py tests/test_operations.py; do test ! -e "/Users/openagi/code/bigclaw-workspaces/BIG-GO-1605/$path" && printf 'absent %s\n' "$path"; done`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1605/bigclaw-go && go test -count=1 ./cmd/bigclawctl ./internal/reporting ./internal/api`
- `cd /Users/openagi/code/bigclaw-workspaces/BIG-GO-1605/bigclaw-go && go test -count=1 ./internal/regression -run TestBIGGO1605`
- Record the exact command lines and pass/fail results in `reports/BIG-GO-1605-validation.md`.
