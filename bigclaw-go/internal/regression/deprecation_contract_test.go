package regression

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestLegacyMainlineCompatibilityManifestStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	manifestPath := filepath.Join(repoRoot, "docs", "reports", "legacy-mainline-compatibility-manifest.json")

	var manifest struct {
		Status                  string `json:"status"`
		Guidance                string `json:"guidance"`
		RetiredPackagingSurface []struct {
			Surface     string `json:"surface"`
			Replacement string `json:"replacement"`
			Status      string `json:"status"`
		} `json:"retired_packaging_surfaces"`
		Modules map[string]struct {
			GoMainlineReplacement string `json:"go_mainline_replacement"`
			LegacyMainlineStatus  string `json:"legacy_mainline_status"`
		} `json:"modules"`
	}
	readJSONFile(t, manifestPath, &manifest)

	if manifest.Status != "go-mainline-compatibility-manifest" {
		t.Fatalf("unexpected manifest status: %+v", manifest)
	}
	if !strings.Contains(manifest.Guidance, "sole implementation mainline") || !strings.Contains(manifest.Guidance, "package entrypoints are retired") {
		t.Fatalf("unexpected guidance: %q", manifest.Guidance)
	}
	if len(manifest.RetiredPackagingSurface) != 2 {
		t.Fatalf("unexpected retired packaging surfaces: %+v", manifest.RetiredPackagingSurface)
	}
	expectedRetired := map[string]string{
		"python -m bigclaw":       "bash scripts/ops/bigclawctl",
		"python -m bigclaw serve": "go run ./bigclaw-go/cmd/bigclawd",
	}
	for _, surface := range manifest.RetiredPackagingSurface {
		wantReplacement, ok := expectedRetired[surface.Surface]
		if !ok {
			t.Fatalf("unexpected retired surface: %+v", surface)
		}
		if surface.Replacement != wantReplacement || surface.Status != "retired" {
			t.Fatalf("unexpected retired surface payload: %+v", surface)
		}
	}

	expectedReplacements := map[string]string{
		"legacy_shim": "scripts/ops/bigclawctl",
	}
	for module, want := range expectedReplacements {
		got, ok := manifest.Modules[module]
		if !ok {
			t.Fatalf("manifest missing module %q", module)
		}
		if got.GoMainlineReplacement != want {
			t.Fatalf("module %s replacement = %q, want %q", module, got.GoMainlineReplacement, want)
		}
		if !strings.Contains(got.LegacyMainlineStatus, "sole implementation mainline") || !strings.Contains(got.LegacyMainlineStatus, "wrapper shims remain") {
			t.Fatalf("module %s legacy status missing mainline guidance: %+v", module, got)
		}
	}
}
