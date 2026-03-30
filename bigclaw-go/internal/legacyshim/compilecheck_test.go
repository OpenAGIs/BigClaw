package legacyshim

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFrozenCompileCheckFilesUsesFrozenShimList(t *testing.T) {
	repoRoot := "/repo"
	got := FrozenCompileCheckFiles(repoRoot)
	want := []string{
		filepath.Join(repoRoot, "src/bigclaw/runtime.py"),
		filepath.Join(repoRoot, "src/bigclaw/__main__.py"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected compile-check files: got=%v want=%v", got, want)
	}
}

func TestCompileCheckRunsPyCompileAgainstFrozenShimList(t *testing.T) {
	repoRoot := "/repo"
	var gotName string
	var gotArgs []string
	result, err := compileCheck(repoRoot, "python-custom", func(name string, args ...string) ([]byte, error) {
		gotName = name
		gotArgs = append([]string(nil), args...)
		return []byte("compiled"), nil
	})
	if err != nil {
		t.Fatalf("compileCheck returned error: %v", err)
	}
	if gotName != "python-custom" {
		t.Fatalf("unexpected python binary: %s", gotName)
	}
	wantArgs := []string{
		"-m",
		"py_compile",
		filepath.Join(repoRoot, "src/bigclaw/runtime.py"),
		filepath.Join(repoRoot, "src/bigclaw/__main__.py"),
	}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("unexpected args: got=%v want=%v", gotArgs, wantArgs)
	}
	if result.Output != "compiled" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
}

func TestCompileCheckReturnsCompilerOutputOnFailure(t *testing.T) {
	expectedErr := errors.New("compile failed")
	result, err := compileCheck("/repo", "python3", func(name string, args ...string) ([]byte, error) {
		return []byte("syntax error"), expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if result.Output != "syntax error" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
}
