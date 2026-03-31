package legacyshim

import (
	"errors"
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

func TestCompileCheckReturnsSuccessWithoutInvokingPythonWhenFrozenListIsEmpty(t *testing.T) {
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
		t.Fatal("expected empty frozen list to skip python invocation")
	}
	if result.Python != "python-custom" {
		t.Fatalf("unexpected python binary: %s", result.Python)
	}
	if len(result.Files) != 0 {
		t.Fatalf("expected no frozen files, got %v", result.Files)
	}
	if result.Output != "" {
		t.Fatalf("expected no compiler output, got %q", result.Output)
	}
}

func TestCompileCheckReturnsCompilerOutputOnFailure(t *testing.T) {
	expectedErr := errors.New("compile failed")
	result, err := compileCheckWithFiles([]string{"/repo/src/bigclaw/runtime.py"}, "python3", func(name string, args ...string) ([]byte, error) {
		return []byte("syntax error"), expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
	if result.Output != "syntax error" {
		t.Fatalf("unexpected output: %q", result.Output)
	}
}

func compileCheckWithFiles(files []string, pythonBin string, run runner) (CompileCheckResult, error) {
	if len(files) == 0 {
		return CompileCheckResult{
			Python: pythonBin,
			Files:  files,
		}, nil
	}
	args := make([]string, 0, len(files)+2)
	args = append(args, "-m", "py_compile")
	args = append(args, files...)
	output, err := run(pythonBin, args...)
	return CompileCheckResult{
		Python: pythonBin,
		Files:  files,
		Output: string(output),
	}, err
}
