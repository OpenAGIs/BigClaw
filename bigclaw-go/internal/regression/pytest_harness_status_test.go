package regression

import (
	"path/filepath"
	"reflect"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestPytestHarnessStatusSnapshotStaysAligned(t *testing.T) {
	root := repoRoot(t)
	reportPath := filepath.Join(root, "docs", "reports", "pytest-harness-status.json")

	var snapshot testharness.PytestHarnessStatusReport
	readJSONFile(t, reportPath, &snapshot)

	live, err := testharness.BuildPytestHarnessStatusReport(filepath.Dir(root))
	if err != nil {
		t.Fatalf("build live pytest harness status report: %v", err)
	}

	if !reflect.DeepEqual(snapshot, live) {
		t.Fatalf("pytest harness status snapshot drifted:\nsnapshot=%+v\nlive=%+v", snapshot, live)
	}
}
