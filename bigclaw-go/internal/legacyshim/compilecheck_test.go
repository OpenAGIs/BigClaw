package legacyshim

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"bigclaw-go/internal/testharness"
)

func TestFrozenCompileCheckFilesUsesFrozenShimList(t *testing.T) {
	repoRoot := "/repo"
	got := FrozenCompileCheckFiles(repoRoot)
	want := []string{
		filepath.Join(repoRoot, "src/bigclaw/service.py"),
		filepath.Join(repoRoot, "src/bigclaw/__main__.py"),
		filepath.Join(repoRoot, "src/bigclaw/legacy_shim.py"),
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
		filepath.Join(repoRoot, "src/bigclaw/service.py"),
		filepath.Join(repoRoot, "src/bigclaw/__main__.py"),
		filepath.Join(repoRoot, "src/bigclaw/legacy_shim.py"),
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

func TestFrozenCompileCheckFilesMatchCheckedInLegacyShimAssets(t *testing.T) {
	projectRoot := testharness.ProjectRoot(t)
	got := FrozenCompileCheckFiles(projectRoot)
	want := []string{
		filepath.Join(projectRoot, "src/bigclaw/service.py"),
		filepath.Join(projectRoot, "src/bigclaw/__main__.py"),
		filepath.Join(projectRoot, "src/bigclaw/legacy_shim.py"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected checked-in compile-check files: got=%v want=%v", got, want)
	}
	for _, path := range got {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat legacy shim asset %s: %v", path, err)
		}
		if info.IsDir() {
			t.Fatalf("expected file path, got directory: %s", path)
		}
	}
}
