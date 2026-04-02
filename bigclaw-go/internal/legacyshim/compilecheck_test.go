package legacyshim

import (
	"errors"
	"reflect"
	"testing"
)

func TestFrozenCompileCheckFilesUsesFrozenShimList(t *testing.T) {
	got := FrozenCompileCheckFiles("/repo")
	want := []string{}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected compile-check files: got=%v want=%v", got, want)
	}
}

func TestCompileCheckSkipsPyCompileWhenFrozenShimListIsEmpty(t *testing.T) {
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
		t.Fatal("expected compileCheck not to invoke python when no retired wrapper files remain")
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

func TestCompileCheckReturnsCompilerOutputOnFailure(t *testing.T) {
	expectedErr := errors.New("compile failed")
	result, err := compileCheckFiles("python3", []string{"/repo/src/bigclaw/__init__.py"}, func(name string, args ...string) ([]byte, error) {
		return []byte("syntax error"), expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if result.Output != "syntax error" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
}
