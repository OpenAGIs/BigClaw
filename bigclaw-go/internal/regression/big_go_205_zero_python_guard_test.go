package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO205ResidualPythonToolingConfigStaysAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	if _, err := os.Stat(filepath.Join(rootRepo, ".pre-commit-config.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected residual Python tooling config to remain absent: .pre-commit-config.yaml")
	}

	readme := readRepoFile(t, rootRepo, "README.md")
	for _, forbidden := range []string{"pre-commit run --all-files", "pre-commit", "ruff"} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("README should not reference residual Python tooling %q", forbidden)
		}
	}
}

func TestBIGGO205RootGoHelperSurfaceRemainsAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		"Makefile",
		"scripts/dev_bootstrap.sh",
		"scripts/ops/bigclawctl",
		"bigclaw-go/cmd/bigclawctl/main.go",
	}
	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, relativePath)); err != nil {
			t.Fatalf("expected retained Go/shell helper path to exist: %s (%v)", relativePath, err)
		}
	}

	readme := readRepoFile(t, rootRepo, "README.md")
	for _, needle := range []string{
		"git diff --check",
		"bash scripts/ops/bigclawctl github-sync --help >/dev/null",
		"make test",
		"make build",
	} {
		if !strings.Contains(readme, needle) {
			t.Fatalf("README missing retained Go/shell helper guidance %q", needle)
		}
	}
}

func TestBIGGO205LaneReportCapturesToolingSweep(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-205-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-205",
		"`.pre-commit-config.yaml`: absent",
		"`README.md` no longer documents `pre-commit run --all-files`",
		"`README.md` now points repository hygiene at `git diff --check` and `bash scripts/ops/bigclawctl github-sync --help >/dev/null`",
		"`Makefile`",
		"`scripts/dev_bootstrap.sh`",
		"`scripts/ops/bigclawctl`",
		"`test ! -e .pre-commit-config.yaml`",
		"`rg -n \"pre-commit|ruff\" README.md`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO205(ResidualPythonToolingConfigStaysAbsent|RootGoHelperSurfaceRemainsAvailable|LaneReportCapturesToolingSweep)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
