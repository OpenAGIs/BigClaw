package legacyshim

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
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

func TestFrozenEntrypointFilesUsesExpectedFreezeAuditList(t *testing.T) {
	repoRoot := "/repo"
	got := FrozenEntrypointFiles(repoRoot)
	want := []string{
		filepath.Join(repoRoot, "src/bigclaw/__init__.py"),
		filepath.Join(repoRoot, "src/bigclaw/__main__.py"),
		filepath.Join(repoRoot, "src/bigclaw/service.py"),
		filepath.Join(repoRoot, "src/bigclaw/legacy_shim.py"),
		filepath.Join(repoRoot, "src/bigclaw/runtime.py"),
		filepath.Join(repoRoot, "src/bigclaw/scheduler.py"),
		filepath.Join(repoRoot, "src/bigclaw/workflow.py"),
		filepath.Join(repoRoot, "src/bigclaw/queue.py"),
		filepath.Join(repoRoot, "src/bigclaw/orchestration.py"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected freeze-audit files: got=%v want=%v", got, want)
	}
}

func TestFreezeAuditReportsMissingReadme(t *testing.T) {
	repoRoot := t.TempDir()
	rootDir := filepath.Join(repoRoot, "src/bigclaw")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("mkdir root dir: %v", err)
	}
	for _, file := range FrozenEntrypointFiles(repoRoot) {
		if err := os.WriteFile(file, []byte(`"""frozen"""`), 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}

	result, err := FreezeAudit(repoRoot)
	if err != nil {
		t.Fatalf("freeze audit returned error: %v", err)
	}
	if !reflect.DeepEqual(result.MissingFiles, []string{filepath.Join(repoRoot, "src/bigclaw/README.md")}) {
		t.Fatalf("unexpected missing files: %v", result.MissingFiles)
	}
}

func TestFreezeAuditReportsMissingMarkers(t *testing.T) {
	repoRoot := t.TempDir()
	rootDir := filepath.Join(repoRoot, "src/bigclaw")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("mkdir root dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rootDir, "README.md"), []byte("frozen tree"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	for _, file := range FrozenEntrypointFiles(repoRoot) {
		body := []byte(`"""frozen compatibility"""`)
		if strings.HasSuffix(file, "__main__.py") {
			body = []byte("print('active')\n")
		}
		if err := os.WriteFile(file, body, 0o644); err != nil {
			t.Fatalf("write %s: %v", file, err)
		}
	}
	if err := os.WriteFile(filepath.Join(rootDir, "models.py"), []byte("class X: pass\n"), 0o644); err != nil {
		t.Fatalf("write inventory file: %v", err)
	}

	result, err := FreezeAudit(repoRoot)
	if err != nil {
		t.Fatalf("freeze audit returned error: %v", err)
	}
	wantMissing := []string{filepath.Join(repoRoot, "src/bigclaw/__main__.py")}
	if !reflect.DeepEqual(result.MissingMarkers, wantMissing) {
		t.Fatalf("unexpected missing markers: got=%v want=%v", result.MissingMarkers, wantMissing)
	}
	if len(result.Files) != len(FrozenEntrypointFiles(repoRoot))+1 {
		t.Fatalf("unexpected file inventory size: %d", len(result.Files))
	}
}
