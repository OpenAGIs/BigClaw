package liveshadowscorecard

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildReportMatchesCanonicalArtifactAtReferenceTime(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	generatedAt, err := time.Parse(time.RFC3339Nano, "2026-03-16T15:58:21.282621Z")
	if err != nil {
		t.Fatalf("parse generated_at: %v", err)
	}

	report, err := BuildReport(BuildOptions{
		RepoRoot:    repoRoot,
		GeneratedAt: generatedAt,
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
	}

	body, err := os.ReadFile(filepath.Join(repoRoot, "bigclaw-go", "docs", "reports", "live-shadow-mirror-scorecard.json"))
	if err != nil {
		t.Fatalf("read canonical artifact: %v", err)
	}
	var canonical map[string]any
	if err := json.Unmarshal(body, &canonical); err != nil {
		t.Fatalf("unmarshal canonical artifact: %v", err)
	}

	generatedJSON, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal generated report: %v", err)
	}
	canonicalJSON, err := json.Marshal(canonical)
	if err != nil {
		t.Fatalf("marshal canonical report: %v", err)
	}
	if string(generatedJSON) != string(canonicalJSON) {
		t.Fatalf("generated report drifted from canonical artifact")
	}
}
