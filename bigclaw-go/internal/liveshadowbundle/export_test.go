package liveshadowbundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExportMatchesCanonicalArtifactsAtReferenceTimes(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	goRoot := filepath.Join(repoRoot, "bigclaw-go")
	tempRoot := t.TempDir()

	copyFixtureFile(t, filepath.Join(goRoot, shadowCompareReport), filepath.Join(tempRoot, shadowCompareReport))
	copyFixtureFile(t, filepath.Join(goRoot, shadowMatrixReport), filepath.Join(tempRoot, shadowMatrixReport))
	copyFixtureFile(t, filepath.Join(goRoot, shadowScorecardReport), filepath.Join(tempRoot, shadowScorecardReport))
	copyFixtureFile(t, filepath.Join(goRoot, rollbackTriggerReport), filepath.Join(tempRoot, rollbackTriggerReport))

	generatedAt, err := time.Parse(time.RFC3339Nano, "2026-03-17T02:35:33.529497Z")
	if err != nil {
		t.Fatalf("parse generatedAt: %v", err)
	}
	rollupGeneratedAt, err := time.Parse(time.RFC3339Nano, "2026-03-17T02:35:33.537339Z")
	if err != nil {
		t.Fatalf("parse rollupGeneratedAt: %v", err)
	}

	if _, err := Export(ExportOptions{
		GoRoot:            tempRoot,
		RunID:             "20260313T085655Z",
		GeneratedAt:       generatedAt,
		RollupGeneratedAt: rollupGeneratedAt,
	}); err != nil {
		t.Fatalf("Export returned error: %v", err)
	}

	assertJSONMatchesCanonical(t, filepath.Join(tempRoot, "docs/reports/live-shadow-summary.json"), filepath.Join(goRoot, "docs/reports/live-shadow-summary.json"))
	assertJSONMatchesCanonical(t, filepath.Join(tempRoot, "docs/reports/live-shadow-index.json"), filepath.Join(goRoot, "docs/reports/live-shadow-index.json"))
	assertJSONMatchesCanonical(t, filepath.Join(tempRoot, "docs/reports/live-shadow-drift-rollup.json"), filepath.Join(goRoot, "docs/reports/live-shadow-drift-rollup.json"))
	assertJSONMatchesCanonical(t, filepath.Join(tempRoot, "docs/reports/live-shadow-runs/20260313T085655Z/summary.json"), filepath.Join(goRoot, "docs/reports/live-shadow-runs/20260313T085655Z/summary.json"))

	assertTextMatchesCanonical(t, filepath.Join(tempRoot, "docs/reports/live-shadow-index.md"), filepath.Join(goRoot, "docs/reports/live-shadow-index.md"))
	assertTextMatchesCanonical(t, filepath.Join(tempRoot, "docs/reports/live-shadow-runs/20260313T085655Z/README.md"), filepath.Join(goRoot, "docs/reports/live-shadow-runs/20260313T085655Z/README.md"))
}

func copyFixtureFile(t *testing.T, source, destination string) {
	t.Helper()
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read fixture %s: %v", source, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", destination, err)
	}
	if err := os.WriteFile(destination, body, 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", destination, err)
	}
}

func assertJSONMatchesCanonical(t *testing.T, generatedPath, canonicalPath string) {
	t.Helper()
	generatedBody, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read generated json %s: %v", generatedPath, err)
	}
	canonicalBody, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("read canonical json %s: %v", canonicalPath, err)
	}
	var generated any
	if err := json.Unmarshal(generatedBody, &generated); err != nil {
		t.Fatalf("unmarshal generated json %s: %v", generatedPath, err)
	}
	var canonical any
	if err := json.Unmarshal(canonicalBody, &canonical); err != nil {
		t.Fatalf("unmarshal canonical json %s: %v", canonicalPath, err)
	}
	generatedJSON, err := json.Marshal(generated)
	if err != nil {
		t.Fatalf("marshal generated json %s: %v", generatedPath, err)
	}
	canonicalJSON, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("marshal canonical json %s: %v", canonicalPath, err)
	}
	if string(generatedJSON) != string(canonicalJSON) {
		t.Fatalf("generated json %s drifted from canonical %s", generatedPath, canonicalPath)
	}
}

func assertTextMatchesCanonical(t *testing.T, generatedPath, canonicalPath string) {
	t.Helper()
	generatedBody, err := os.ReadFile(generatedPath)
	if err != nil {
		t.Fatalf("read generated text %s: %v", generatedPath, err)
	}
	canonicalBody, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("read canonical text %s: %v", canonicalPath, err)
	}
	if string(generatedBody) != string(canonicalBody) {
		t.Fatalf("generated text %s drifted from canonical %s", generatedPath, canonicalPath)
	}
}
