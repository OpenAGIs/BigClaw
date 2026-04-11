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
		Status   string `json:"status"`
		Guidance string `json:"guidance"`
		Modules  map[string]struct {
			GoMainlineReplacement string `json:"go_mainline_replacement"`
			LegacyMainlineStatus  string `json:"legacy_mainline_status"`
		} `json:"modules"`
	}
	readJSONFile(t, manifestPath, &manifest)

	if manifest.Status != "go-mainline-compatibility-manifest" {
		t.Fatalf("unexpected manifest status: %+v", manifest)
	}
	if !strings.Contains(manifest.Guidance, "sole implementation mainline") || !strings.Contains(manifest.Guidance, "migration-only") {
		t.Fatalf("unexpected guidance: %q", manifest.Guidance)
	}

	expectedReplacements := map[string]string{
		"runtime":       "bigclaw-go/internal/worker/runtime.go",
		"scheduler":     "bigclaw-go/internal/scheduler/scheduler.go",
		"workflow":      "bigclaw-go/internal/workflow/engine.go",
		"orchestration": "bigclaw-go/internal/workflow/orchestration.go",
		"queue":         "bigclaw-go/internal/queue/queue.go",
	}
	for module, want := range expectedReplacements {
		got, ok := manifest.Modules[module]
		if !ok {
			t.Fatalf("manifest missing module %q", module)
		}
		if got.GoMainlineReplacement != want {
			t.Fatalf("module %s replacement = %q, want %q", module, got.GoMainlineReplacement, want)
		}
		if !strings.Contains(got.LegacyMainlineStatus, "sole implementation mainline") {
			t.Fatalf("module %s legacy status missing mainline guidance: %+v", module, got)
		}
	}
}
