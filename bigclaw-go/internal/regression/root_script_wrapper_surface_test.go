package regression

import (
	"strings"
	"testing"
)

func TestRootScriptWrappersStayGoFirst(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	bigclawctl := readRepoFile(t, repoRoot, "scripts/ops/bigclawctl")
	requiredBigclawctl := []string{
		`cd "$repo_root/bigclaw-go"`,
		`exec go run ./cmd/bigclawctl "${out_args[@]}" --repo "$repo_root"`,
		`exec go run ./cmd/bigclawctl "${out_args[@]}"`,
	}
	for _, needle := range requiredBigclawctl {
		if !strings.Contains(bigclawctl, needle) {
			t.Fatalf("scripts/ops/bigclawctl missing Go-first wrapper behavior %q", needle)
		}
	}
	disallowedBigclawctl := []string{
		"create_issues.py",
		"dev_smoke.py",
		"bigclaw_github_sync.py",
		"bigclaw_refill_queue.py",
		"symphony_workspace_bootstrap.py",
		"symphony_workspace_validate.py",
		"python3 ",
	}
	for _, needle := range disallowedBigclawctl {
		if strings.Contains(bigclawctl, needle) {
			t.Fatalf("scripts/ops/bigclawctl should not reference retired Python surfaces %q", needle)
		}
	}

	symphony := readRepoFile(t, repoRoot, "scripts/ops/bigclaw-symphony")
	requiredSymphony := []string{
		`exec bash "$script_dir/bigclawctl" symphony "$@"`,
	}
	for _, needle := range requiredSymphony {
		if !strings.Contains(symphony, needle) {
			t.Fatalf("scripts/ops/bigclaw-symphony missing Go-first delegation %q", needle)
		}
	}
	disallowedSymphony := []string{
		"symphony_workspace_bootstrap.py",
		"symphony_workspace_validate.py",
		"python3 ",
	}
	for _, needle := range disallowedSymphony {
		if strings.Contains(symphony, needle) {
			t.Fatalf("scripts/ops/bigclaw-symphony should not reference retired Python surfaces %q", needle)
		}
	}

	devBootstrap := readRepoFile(t, repoRoot, "scripts/dev_bootstrap.sh")
	requiredDevBootstrap := []string{
		`go test ./cmd/bigclawctl`,
		`if [ "${BIGCLAW_ENABLE_LEGACY_PYTHON:-0}" = "1" ]; then`,
		`bash "$repo_root/scripts/ops/bigclawctl" dev-smoke`,
		`go test ./internal/bootstrap`,
		`bash "$repo_root/scripts/ops/bigclawctl" legacy-python compile-check --repo "$repo_root" --python python3 --json`,
	}
	for _, needle := range requiredDevBootstrap {
		if !strings.Contains(devBootstrap, needle) {
			t.Fatalf("scripts/dev_bootstrap.sh missing expected Go-first bootstrap behavior %q", needle)
		}
	}
	disallowedDevBootstrap := []string{
		"scripts/dev_smoke.py",
		"scripts/create_issues.py",
		"bigclaw_workspace_bootstrap.py",
		"symphony_workspace_bootstrap.py",
		"symphony_workspace_validate.py",
	}
	for _, needle := range disallowedDevBootstrap {
		if strings.Contains(devBootstrap, needle) {
			t.Fatalf("scripts/dev_bootstrap.sh should not reference retired Python entrypoints %q", needle)
		}
	}
}
