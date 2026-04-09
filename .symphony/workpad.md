# BIG-GO-187 Workpad

Date: 2026-04-09

## Plan

1. Re-baseline active repo directories for residual legacy Python references and keep the issue scoped to live migration docs plus a focused regression/report lane.
2. Reduce repeated `.py` path enumerations in the active cutover and CLI migration docs by collapsing them into compact grouped references while preserving the migration record and Go replacement pointers.
3. Add `BIG-GO-187` regression and report artifacts that pin the reduced residual-reference surface and document the broad-sweep validation evidence.
4. Run targeted validation, record exact commands and results, then commit and push the branch.

## Acceptance

- `.symphony/workpad.md` contains the issue plan, acceptance criteria, and exact validation commands before any code edits land.
- Active migration docs reduce legacy Python reference density without changing the documented Go ownership or operator replacement guidance.
- `BIG-GO-187` artifacts document the sweep and a regression test fails if the compacted doc surface regresses back to the previous verbose legacy `.py` lists.
- Targeted validation commands pass and their exact commands/results are captured in the issue validation artifact.
- Changes stay scoped to this issue's documentation and regression/reporting surface.

## Validation

- `for f in docs/go-cli-script-migration-plan.md docs/go-mainline-cutover-issue-pack.md docs/go-mainline-cutover-handoff.md; do before=$(git show HEAD:"$f" | rg -o 'src/bigclaw/[^`[:space:]]+|scripts/[^`[:space:]]+\.py|bigclaw-go/scripts/[^`[:space:]]+\.py|python3|\.py' | wc -l | tr -d ' '); after=$(rg -o 'src/bigclaw/[^`[:space:]]+|scripts/[^`[:space:]]+\.py|bigclaw-go/scripts/[^`[:space:]]+\.py|python3|\.py' "$f" | wc -l | tr -d ' '); printf '%s %s %s\n' "$f" "$before" "$after"; done`
- `find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`
- `find docs bigclaw-go/internal bigclaw-go/cmd -type f -name '*.py' -print 2>/dev/null | sort`
- `rg -n "python3|\\.py\\b|#!/usr/bin/env python|#!/usr/bin/python" docs bigclaw-go/internal bigclaw-go/cmd --glob '!bigclaw-go/internal/regression/**' --glob '!bigclaw-go/docs/reports/**' | head -n 200`
- `cd bigclaw-go && go test ./internal/regression -run 'TestBIGGO187' -count=1`
- `git status --short`
