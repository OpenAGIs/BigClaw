package regression

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"bigclaw-go/internal/legacyshim"
)

func TestTopLevelModulePurgeTranche30(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	pythonFiles := relativePythonFiles(t, repoRoot)
	wantPythonFiles := []string{
		"src/bigclaw/__init__.py",
	}
	if !reflect.DeepEqual(pythonFiles, wantPythonFiles) {
		t.Fatalf("unexpected remaining python files: got=%v want=%v", pythonFiles, wantPythonFiles)
	}

	wantCompileCheck := []string{
		filepath.Join(repoRoot, "src/bigclaw/__init__.py"),
	}
	if got := legacyshim.FrozenCompileCheckFiles(repoRoot); !reflect.DeepEqual(got, wantCompileCheck) {
		t.Fatalf("unexpected compile-check files: got=%v want=%v", got, wantCompileCheck)
	}
}

func TestTopLevelModulePurgeTranche30ImportSurface(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	cmd := exec.Command(
		"python3",
		"-S",
		"-c",
		`import json
import bigclaw
print(json.dumps({
    "task": getattr(bigclaw, "Task").__name__,
    "priority_p0": getattr(bigclaw, "Priority").P0.value,
    "risk_level_high": getattr(bigclaw, "RiskLevel").HIGH.value,
    "has_report": hasattr(bigclaw, "render_scope_freeze_report"),
}))`,
	)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(), "PYTHONPATH="+filepath.Join(repoRoot, "src"))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("isolated python import bigclaw: %v\n%s", err, output)
	}

	var got struct {
		Task          string `json:"task"`
		PriorityP0    int    `json:"priority_p0"`
		RiskLevelHigh string `json:"risk_level_high"`
		HasReport     bool   `json:"has_report"`
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatalf("decode isolated python output: %v\n%s", err, output)
	}

	if got.Task != "Task" || got.PriorityP0 != 0 || got.RiskLevelHigh != "high" || !got.HasReport {
		t.Fatalf("unexpected isolated import surface: %+v", got)
	}
}

func relativePythonFiles(t *testing.T, repoRoot string) []string {
	t.Helper()

	var files []string
	err := filepath.Walk(filepath.Join(repoRoot, "src"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".py" {
			return nil
		}
		relativePath, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(relativePath))
		return nil
	})
	if err != nil {
		t.Fatalf("walk python files: %v", err)
	}
	sort.Strings(files)
	return files
}
