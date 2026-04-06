Issue: `BIG-GO-1483`

Final branch head: `eb93ecfc1c80cbaff7afc45c204ab17f9a13579d`

Summary:
- `bigclaw-go/scripts` remains physically Python-free before and after this lane.
- The remaining checked-in caller references to retired `bigclaw-go/scripts` Python entrypoints were removed from the live migration plan.
- The active `bigclaw-go/scripts` surface is now documented and guarded as Go CLI, shell, or Go helper entrypoints only.

Before/after evidence:
- Repository `.py` files before: `0`
- Repository `.py` files after: `0`
- `bigclaw-go/scripts/*.py` before: `0`
- `bigclaw-go/scripts/*.py` after: `0`
- Checked-in caller references to retired `bigclaw-go/scripts` Python entrypoints before: `23`
- Checked-in caller references to retired `bigclaw-go/scripts` Python entrypoints after: `0`

Validation:
- `git show a63c8ec0f999d976a1af890c920a54ac2d6c693a:docs/go-cli-script-migration-plan.md | rg -n 'bigclaw-go/scripts/.*\.py' | wc -l | tr -d ' '` -> `23`
- `find . -path '*/.git' -prune -o -name '*.py' -type f -print | sort` -> no output
- `find bigclaw-go/scripts -type f -name '*.py' | sort` -> no output
- `rg -n --glob '!reports/**' --glob '!bigclaw-go/docs/reports/**' --glob '!local-issues.json' --glob '!bigclaw-go/internal/regression/**' --glob '!.symphony/**' 'bigclaw-go/scripts/.*\.py' README.md docs scripts .github bigclaw-go | sort` -> no output
- `cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1160|TestBIGGO1483|TestE2E'` -> `ok  	bigclaw-go/internal/regression	3.449s`

Blocker:
- The repository already started at zero checked-in Python files, so the issue’s requested physical Python-file reduction could not be satisfied in this branch baseline. The completed scope therefore removes the remaining checked-in caller references and records exact before/after evidence instead.
