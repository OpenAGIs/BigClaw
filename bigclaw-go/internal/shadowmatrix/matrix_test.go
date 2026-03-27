package shadowmatrix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bigclaw-go/internal/shadowcompare"
)

func TestBuildReportMatchesCanonicalArtifact(t *testing.T) {
	t.Parallel()

	repoRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	canonicalPath := filepath.Join(repoRoot, "bigclaw-go/docs/reports/shadow-matrix-report.json")
	body, err := os.ReadFile(canonicalPath)
	if err != nil {
		t.Fatalf("read canonical artifact: %v", err)
	}
	var canonical map[string]any
	if err := json.Unmarshal(body, &canonical); err != nil {
		t.Fatalf("unmarshal canonical artifact: %v", err)
	}

	resultBySource := map[string]map[string]any{}
	for _, result := range mapSliceAt(canonical, "results") {
		source := stringValue(result["source_file"], "")
		resultBySource[source] = result
		resultBySource[filepath.Join(repoRoot, "bigclaw-go", strings.TrimPrefix(source, "./"))] = result
	}

	report, err := BuildReport(BuildOptions{
		TaskFiles: []string{
			filepath.Join(repoRoot, "bigclaw-go/examples/shadow-task.json"),
			filepath.Join(repoRoot, "bigclaw-go/examples/shadow-task-budget.json"),
			filepath.Join(repoRoot, "bigclaw-go/examples/shadow-task-validation.json"),
		},
		CorpusManifestPath: filepath.Join(repoRoot, "bigclaw-go/examples/shadow-corpus-manifest.json"),
		Compare: func(opts shadowcompare.CompareOptions) (map[string]any, error) {
			source := stringValue(opts.Task["_source_file"], "")
			result := cloneMap(resultBySource[source])
			return result, nil
		},
	})
	if err != nil {
		t.Fatalf("BuildReport returned error: %v", err)
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
