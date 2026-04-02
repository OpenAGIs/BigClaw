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
		Status         string `json:"status"`
		Guidance       string `json:"guidance"`
		RuntimeWarning struct {
			Surface     string `json:"surface"`
			Replacement string `json:"replacement"`
			Message     string `json:"message"`
		} `json:"runtime_warning"`
		ServiceWarning struct {
			Surface     string `json:"surface"`
			Replacement string `json:"replacement"`
			Message     string `json:"message"`
		} `json:"service_warning"`
		Modules map[string]struct {
			GoMainlineReplacement string `json:"go_mainline_replacement"`
			LegacyMainlineStatus  string `json:"legacy_mainline_status"`
		} `json:"modules"`
	}
	readJSONFile(t, manifestPath, &manifest)

	if manifest.Status != "go-mainline-compatibility-manifest" {
		t.Fatalf("unexpected manifest status: %+v", manifest)
	}
	if !strings.Contains(manifest.Guidance, "sole implementation mainline") ||
		!strings.Contains(manifest.Guidance, "migration-only") ||
		!strings.Contains(manifest.Guidance, "src/bigclaw/__init__.py") {
		t.Fatalf("unexpected guidance: %q", manifest.Guidance)
	}

	if manifest.RuntimeWarning.Surface != "python -m bigclaw" ||
		manifest.RuntimeWarning.Replacement != "bash scripts/ops/bigclawctl" ||
		!strings.Contains(manifest.RuntimeWarning.Message, "frozen for migration-only use") ||
		!strings.Contains(manifest.RuntimeWarning.Message, "src/bigclaw/__init__.py") ||
		!strings.Contains(manifest.RuntimeWarning.Message, "Use bash scripts/ops/bigclawctl instead.") {
		t.Fatalf("unexpected runtime warning payload: %+v", manifest.RuntimeWarning)
	}
	if manifest.ServiceWarning.Surface != "python -m bigclaw serve" ||
		manifest.ServiceWarning.Replacement != "go run ./bigclaw-go/cmd/bigclawd" ||
		!strings.Contains(manifest.ServiceWarning.Message, "frozen for migration-only use") ||
		!strings.Contains(manifest.ServiceWarning.Message, "src/bigclaw/__init__.py") ||
		!strings.Contains(manifest.ServiceWarning.Message, "Use go run ./bigclaw-go/cmd/bigclawd instead.") {
		t.Fatalf("unexpected service warning payload: %+v", manifest.ServiceWarning)
	}

	expectedReplacements := map[string]string{
		"runtime":       "bigclaw-go/internal/worker/runtime.go",
		"scheduler":     "bigclaw-go/internal/scheduler/scheduler.go",
		"workflow":      "bigclaw-go/internal/workflow/engine.go",
		"orchestration": "bigclaw-go/internal/workflow/orchestration.go",
		"queue":         "bigclaw-go/internal/queue/queue.go",
		"service":       "bigclaw-go/cmd/bigclawd/main.go",
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
		if module != "service" && !strings.Contains(got.LegacyMainlineStatus, "src/bigclaw/__init__.py") {
			t.Fatalf("module %s legacy status missing package-root compatibility file guidance: %+v", module, got)
		}
		if module == "service" {
			if strings.Contains(got.LegacyMainlineStatus, "service.py remains migration-only compatibility scaffolding") {
				t.Fatalf("module %s legacy status still references deleted service.py: %+v", module, got)
			}
			if !strings.Contains(got.LegacyMainlineStatus, "package-root service compatibility surface remains migration-only") {
				t.Fatalf("module %s legacy status missing package-root compatibility guidance: %+v", module, got)
			}
		}
	}
}
