package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO174CompatibilityWrappersStayThinPassThroughs(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	cases := map[string]string{
		"scripts/ops/bigclaw-issue":    `exec "$script_dir/bigclawctl" issue "$@"`,
		"scripts/ops/bigclaw-panel":    `exec "$script_dir/bigclawctl" panel "$@"`,
		"scripts/ops/bigclaw-symphony": `exec "$script_dir/bigclawctl" symphony "$@"`,
	}

	for relativePath, expectedExec := range cases {
		body := readRepoFile(t, rootRepo, relativePath)
		if strings.Contains(body, `exec bash "$script_dir/bigclawctl"`) {
			t.Fatalf("%s should exec the shared wrapper directly without nesting bash: %s", relativePath, body)
		}
		if !strings.Contains(body, expectedExec) {
			t.Fatalf("%s missing direct shared-wrapper exec %q: %s", relativePath, expectedExec, body)
		}
	}
}

func TestBIGGO174BootstrapUsesDirectRootEntrypoint(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	body := readRepoFile(t, rootRepo, "scripts/dev_bootstrap.sh")

	if strings.Contains(body, `bash "$repo_root/scripts/ops/bigclawctl" dev-smoke`) {
		t.Fatalf("scripts/dev_bootstrap.sh should call the root Go entrypoint directly: %s", body)
	}
	if !strings.Contains(body, `"$repo_root/scripts/ops/bigclawctl" dev-smoke`) {
		t.Fatalf("scripts/dev_bootstrap.sh missing direct dev-smoke invocation: %s", body)
	}
}

func TestBIGGO174ReadmePositionsCompatibilityWrappersAsSecondary(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	readme := readRepoFile(t, rootRepo, "README.md")

	required := []string{
		"thin compatibility pass-throughs",
		"single supported root",
		"operator entrypoint",
		"`scripts/ops/bigclawctl`",
	}
	for _, needle := range required {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README.md missing BIG-GO-174 wrapper posture guidance %q", needle)
		}
	}

	if strings.Contains(readme, "retained as compatibility wrappers, but the preferred operator path is now `scripts/ops/bigclawctl`.") {
		t.Fatalf("README.md should use the tightened BIG-GO-174 guidance for compatibility wrappers")
	}

	for _, relativePath := range []string{
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
	} {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected compatibility wrapper to remain present: %s (%v)", relativePath, err)
		}
	}
}
