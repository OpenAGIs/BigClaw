package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBIGGO1609RepositoryHasNoPythonFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}
}

func TestBIGGO1609PackageBootstrapGluePathsRemainAbsent(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retiredPaths := []string{
		"src/bigclaw/__init__.py",
		"src/bigclaw/__main__.py",
		"src/bigclaw/legacy_shim.py",
		"src/bigclaw/workspace_bootstrap_cli.py",
	}

	for _, relativePath := range retiredPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); !os.IsNotExist(err) {
			t.Fatalf("expected package bootstrap glue to remain absent: %s (err=%v)", relativePath, err)
		}
	}
}

func TestBIGGO1609GoNativeBootstrapSurfacesRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	replacementPaths := []string{
		"bigclaw-go/cmd/bigclawctl/main.go",
		"bigclaw-go/cmd/bigclawd/main.go",
		"bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go",
		"bigclaw-go/internal/bootstrap/bootstrap.go",
		"bigclaw-go/internal/bootstrap/bootstrap_test.go",
		"bigclaw-go/internal/api/broker_bootstrap_surface.go",
		"bigclaw-go/internal/api/broker_bootstrap_surface_test.go",
		"bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json",
		"scripts/ops/bigclawctl",
	}

	for _, relativePath := range replacementPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected Go/native bootstrap surface to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO1609LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-1609-package-bootstrap-glue-sweep.md")

	for _, needle := range []string{
		"BIG-GO-1609",
		"Repository-wide Python file count: `0`.",
		"`src/bigclaw/__init__.py`",
		"`src/bigclaw/__main__.py`",
		"`src/bigclaw/legacy_shim.py`",
		"`src/bigclaw/workspace_bootstrap_cli.py`",
		"`bigclaw-go/cmd/bigclawctl/main.go`",
		"`bigclaw-go/cmd/bigclawd/main.go`",
		"`bigclaw-go/cmd/bigclawctl/legacy_shim_help_test.go`",
		"`bigclaw-go/internal/bootstrap/bootstrap.go`",
		"`bigclaw-go/internal/api/broker_bootstrap_surface.go`",
		"`bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json`",
		"`scripts/ops/bigclawctl`",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`for path in src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/legacy_shim.py src/bigclaw/workspace_bootstrap_cli.py; do test ! -e \"$path\" || echo \"present: $path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1609(RepositoryHasNoPythonFiles|PackageBootstrapGluePathsRemainAbsent|GoNativeBootstrapSurfacesRemainAvailable|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche7$|TestTopLevelModulePurgeTranche17$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
