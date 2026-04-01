package legacyshim

import (
	"reflect"
	"testing"
)

func TestFrozenCompileCheckFilesReturnsNoRetiredPythonShims(t *testing.T) {
	repoRoot := "/repo"
	got := FrozenCompileCheckFiles(repoRoot)
	want := []string{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected compile-check files: got=%v want=%v", got, want)
	}
}

func TestCompileCheckSkipsPythonInvocationWhenNoFrozenShimFilesRemain(t *testing.T) {
	repoRoot := "/repo"
	called := false
	result, err := compileCheck(repoRoot, "python-custom", func(name string, args ...string) ([]byte, error) {
		called = true
		return []byte("compiled"), nil
	})
	if err != nil {
		t.Fatalf("compileCheck returned error: %v", err)
	}
	if called {
		t.Fatal("expected compileCheck to skip python invocation when no files remain")
	}
	if result.Python != "python-custom" {
		t.Fatalf("unexpected python binary: %s", result.Python)
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected no files, got %v", result.Files)
	}
	if result.Output != "" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
}
