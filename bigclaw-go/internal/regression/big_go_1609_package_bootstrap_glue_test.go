package regression

import (
	"encoding/json"
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

func TestBIGGO1609CompatibilityManifestOmitsRetiredPackageBootstrapGlue(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	type manifestModule struct {
		GoMainlineReplacement string `json:"go_mainline_replacement"`
		LegacyMainlineStatus  string `json:"legacy_mainline_status"`
	}
	type compatibilityManifest struct {
		Guidance string                    `json:"guidance"`
		Modules  map[string]manifestModule `json:"modules"`
	}

	var manifest compatibilityManifest
	if err := json.Unmarshal(
		[]byte(readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/legacy-mainline-compatibility-manifest.json")),
		&manifest,
	); err != nil {
		t.Fatalf("decode compatibility manifest: %v", err)
	}

	forbiddenModuleKeys := []string{
		"__init__",
		"__main__",
		"legacy_shim",
		"workspace_bootstrap_cli",
	}
	for _, moduleKey := range forbiddenModuleKeys {
		if _, exists := manifest.Modules[moduleKey]; exists {
			t.Fatalf("expected compatibility manifest to omit retired bootstrap glue module key %q", moduleKey)
		}
	}

	retiredPathFragments := []string{
		"src/bigclaw/__init__.py",
		"src/bigclaw/__main__.py",
		"src/bigclaw/legacy_shim.py",
		"src/bigclaw/workspace_bootstrap_cli.py",
	}
	for _, fragment := range retiredPathFragments {
		if strings.Contains(manifest.Guidance, fragment) {
			t.Fatalf("expected compatibility manifest guidance to stay clear of retired package bootstrap glue %q", fragment)
		}
	}

	for moduleKey, module := range manifest.Modules {
		if !strings.HasPrefix(module.GoMainlineReplacement, "bigclaw-go/") {
			t.Fatalf("expected Go-first replacement path for module %q, got %q", moduleKey, module.GoMainlineReplacement)
		}
		for _, fragment := range retiredPathFragments {
			if strings.Contains(module.GoMainlineReplacement, fragment) || strings.Contains(module.LegacyMainlineStatus, fragment) {
				t.Fatalf("expected module %q to stay clear of retired package bootstrap glue %q", moduleKey, fragment)
			}
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
		"Compatibility manifest remains Go-first",
		"package bootstrap glue paths or module keys.",
		"`find . -path '*/.git' -prune -o -type f -name '*.py' -print | sort`",
		"`for path in src/bigclaw/__init__.py src/bigclaw/__main__.py src/bigclaw/legacy_shim.py src/bigclaw/workspace_bootstrap_cli.py; do test ! -e \"$path\" || echo \"present: $path\"; done`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO1609(RepositoryHasNoPythonFiles|PackageBootstrapGluePathsRemainAbsent|GoNativeBootstrapSurfacesRemainAvailable|CompatibilityManifestOmitsRetiredPackageBootstrapGlue|LaneReportCapturesSweepState)$|TestTopLevelModulePurgeTranche7$|TestTopLevelModulePurgeTranche17$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}
