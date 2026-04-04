package regression

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1176ResidualScriptSurfacesStayPythonFree(t *testing.T) {
	goRepoRoot := repoRoot(t)
	rootRepo := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	requiredPaths := []string{
		filepath.Join(rootRepo, "scripts", "dev_bootstrap.sh"),
		filepath.Join(rootRepo, "scripts", "ops", "bigclawctl"),
		filepath.Join(rootRepo, "scripts", "ops", "bigclaw-issue"),
		filepath.Join(rootRepo, "scripts", "ops", "bigclaw-panel"),
		filepath.Join(rootRepo, "scripts", "ops", "bigclaw-symphony"),
		filepath.Join(goRepoRoot, "scripts", "benchmark", "run_suite.sh"),
		filepath.Join(goRepoRoot, "scripts", "e2e", "broker_bootstrap_summary.go"),
		filepath.Join(goRepoRoot, "scripts", "e2e", "kubernetes_smoke.sh"),
		filepath.Join(goRepoRoot, "scripts", "e2e", "ray_smoke.sh"),
		filepath.Join(goRepoRoot, "scripts", "e2e", "run_all.sh"),
	}
	for _, path := range requiredPaths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected active BIG-GO-1176 residual script surface to exist: %s (%v)", path, err)
		}
	}

	auditedDirs := []string{
		filepath.Join(rootRepo, "scripts"),
		filepath.Join(goRepoRoot, "scripts"),
	}
	for _, dir := range auditedDirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".py") {
				t.Fatalf("expected BIG-GO-1176 audited script surface to stay Python-free: %s", path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk BIG-GO-1176 audited script surface %s: %v", dir, err)
		}
	}
}

func TestBIGGO1176MigrationDocsCaptureGoOnlyResidualSweep(t *testing.T) {
	goRepoRoot := repoRoot(t)
	rootRepo := filepath.Clean(filepath.Join(goRepoRoot, ".."))

	goDoc := readRepoFile(t, goRepoRoot, "docs/go-cli-script-migration.md")
	rootDoc := readRepoFile(t, rootRepo, "docs/go-cli-script-migration-plan.md")

	requiredGoDoc := []string{
		"Issues: `BIG-GO-902`, `BIG-GO-1053`, `BIG-GO-1160`, `BIG-GO-1176`",
		"## BIG-GO-1176 Residual Surface",
		"`BIG-GO-1176` records the residual live script surface after the repo reached a",
		"zero-`.py` baseline.",
		"`bigclaw-go/scripts/benchmark/run_suite.sh`",
		"`bigclaw-go/scripts/e2e/run_all.sh`",
		"`bigclaw-go/scripts/e2e/kubernetes_smoke.sh`",
		"`bigclaw-go/scripts/e2e/ray_smoke.sh`",
		"`bigclaw-go/scripts/e2e/broker_bootstrap_summary.go`",
		"`go run ./cmd/bigclawctl automation benchmark run-matrix ...`",
		"`go run ./cmd/bigclawctl automation e2e run-task-smoke ...`",
	}
	for _, needle := range requiredGoDoc {
		if !strings.Contains(goDoc, needle) {
			t.Fatalf("bigclaw-go/docs/go-cli-script-migration.md missing BIG-GO-1176 residual sweep evidence %q", needle)
		}
	}

	requiredRootDoc := []string{
		"`BIG-GO-1176` narrows the follow-up evidence to the residual live script surface",
		"that remains under `scripts/` and `bigclaw-go/scripts/`.",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`bash scripts/ops/bigclawctl dev-smoke`",
		"`bash scripts/ops/bigclawctl create-issues ...`",
	}
	for _, needle := range requiredRootDoc {
		if !strings.Contains(rootDoc, needle) {
			t.Fatalf("docs/go-cli-script-migration-plan.md missing BIG-GO-1176 residual sweep evidence %q", needle)
		}
	}
}
