package regression

import (
	"testing"

	"bigclaw-go/internal/legacyshim"
)

func TestBIGGO1195LegacyPythonCompileCheckMatchesZeroPythonBaseline(t *testing.T) {
	rootRepo := repoRoot(t)

	pythonFiles := collectPythonFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free before compile-check validation, found %d file(s): %v", len(pythonFiles), pythonFiles)
	}

	frozenFiles := legacyshim.FrozenCompileCheckFiles(rootRepo)
	if len(frozenFiles) != 0 {
		t.Fatalf("expected legacy-python compile-check to track no frozen shim files, found %v", frozenFiles)
	}

	result, err := legacyshim.CompileCheck(rootRepo, "python3")
	if err != nil {
		t.Fatalf("run legacy-python compile-check: %v", err)
	}
	if result.Python != "python3" {
		t.Fatalf("unexpected python binary: got %q want %q", result.Python, "python3")
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected legacy-python compile-check to report an empty file list, found %v", result.Files)
	}
	if result.Output != "" {
		t.Fatalf("expected legacy-python compile-check to skip python execution for empty file list, got output %q", result.Output)
	}
}
