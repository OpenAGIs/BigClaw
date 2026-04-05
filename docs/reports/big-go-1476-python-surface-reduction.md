# BIG-GO-1476 Python Surface Reduction

## Deleted files

| Deleted path | Go owner or delete condition |
| --- | --- |
| `src/bigclaw/workspace_bootstrap.py` | Replaced by `bigclaw-go/internal/bootstrap/bootstrap.go` |
| `src/bigclaw/workspace_bootstrap_cli.py` | Replaced by `bigclaw-go/cmd/bigclawctl/main.go` (`workspace bootstrap` / `workspace cleanup`) |
| `src/bigclaw/workspace_bootstrap_validation.py` | Replaced by `bigclaw-go/internal/bootstrap/bootstrap.go` (`BuildValidationReport`, `RenderValidationMarkdown`, `WriteValidationReport`) |
| `scripts/ops/bigclaw_workspace_bootstrap.py` | Deleted because operators now call `scripts/ops/bigclawctl workspace bootstrap` directly |
| `scripts/ops/symphony_workspace_bootstrap.py` | Deleted because operators now call `scripts/ops/bigclawctl workspace bootstrap` directly |
| `scripts/ops/symphony_workspace_validate.py` | Deleted because operators now call `scripts/ops/bigclawctl workspace validate` directly |
| `tests/test_workspace_bootstrap.py` | Deleted because the replacement behavior is covered by `bigclaw-go/internal/bootstrap/bootstrap_test.go` and `bigclaw-go/cmd/bigclawctl/main_test.go` |

## Validation summary

- Python file count before change: `138`
- Python file count after change: `131`
- Net reduction: `7`

## Scope note

This issue only removes workspace/bootstrap helper surfaces that are already duplicated in the Go CLI. Other Python files remain out of scope for this refill slice.
