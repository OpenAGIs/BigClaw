package regression

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBigClawGoScriptsStayGoNative(t *testing.T) {
	repoRoot := repoRoot(t)
	scriptsRoot := filepath.Join(repoRoot, "scripts")

	var pythonFiles []string
	var goFiles []string
	err := filepath.WalkDir(scriptsRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		switch filepath.Ext(path) {
		case ".py":
			pythonFiles = append(pythonFiles, filepath.ToSlash(rel))
		case ".go":
			goFiles = append(goFiles, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", scriptsRoot, err)
	}

	sort.Strings(pythonFiles)
	sort.Strings(goFiles)

	if len(pythonFiles) != 0 {
		t.Fatalf("bigclaw-go/scripts must stay free of Python helpers, found %v", pythonFiles)
	}
	if len(goFiles) == 0 {
		t.Fatal("expected at least one Go replacement under bigclaw-go/scripts")
	}
	if !containsPathFold(goFiles, "scripts/e2e/broker_bootstrap_summary.go") {
		t.Fatalf("expected broker bootstrap Go helper replacement, found %v", goFiles)
	}

	for _, relative := range []string{
		"pyproject.toml",
		"setup.py",
		"scripts/pyproject.toml",
		"scripts/setup.py",
	} {
		if _, err := os.Stat(filepath.Join(repoRoot, relative)); err == nil {
			t.Fatalf("%s should not exist after Go script migration", filepath.ToSlash(relative))
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat %s: %v", relative, err)
		}
	}
}

func containsPathFold(items []string, want string) bool {
	for _, item := range items {
		if strings.EqualFold(item, want) {
			return true
		}
	}
	return false
}
