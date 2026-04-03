package legacyshim

import (
	"reflect"
	"testing"
)

func TestFrozenCompileCheckFilesUsesFrozenShimList(t *testing.T) {
	repoRoot := "/repo"
	got := FrozenCompileCheckFiles(repoRoot)
	want := []string{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected compile-check files: got=%v want=%v", got, want)
	}
}

func TestCompileCheckSkipsPythonWhenFrozenShimListIsEmpty(t *testing.T) {
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
		t.Fatal("expected compileCheck not to invoke python when no frozen shim files remain")
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

func TestCompileCheckReturnsEmptyOutputWhenFrozenShimListIsEmpty(t *testing.T) {
	result, err := compileCheck("/repo", "python3", func(name string, args ...string) ([]byte, error) {
		t.Fatal("expected compileCheck not to invoke python when no frozen shim files remain")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("expected empty compile check to skip python and return no error, got %v", err)
	}
	if result.Output != "" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
}
