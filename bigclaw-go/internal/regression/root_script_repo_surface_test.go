package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootScriptRepoSurfacesStayGoOnly(t *testing.T) {
	repoRoot := regressionRepoRoot(t)

	disallowed := []string{
		"python3 scripts/create_issues.py",
		"python3 scripts/dev_smoke.py",
		"python3 scripts/ops/bigclaw_github_sync.py",
		"python3 scripts/ops/bigclaw_refill_queue.py",
		"python3 scripts/ops/bigclaw_workspace_bootstrap.py",
		"python3 scripts/ops/symphony_workspace_bootstrap.py",
		"python3 scripts/ops/symphony_workspace_validate.py",
	}

	skipPaths := map[string]bool{
		".git":                           true,
		"local-issues.json":              true,
		"bigclaw-go/internal/regression": true,
	}

	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(repoRoot, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		for skip := range skipPaths {
			if relative == skip || strings.HasPrefix(relative, skip+"/") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}

		switch filepath.Ext(relative) {
		case ".md", ".json", ".yaml", ".yml", ".txt", ".sh":
		default:
			return nil
		}

		bodyBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		body := string(bodyBytes)
		for _, needle := range disallowed {
			if strings.Contains(body, needle) {
				t.Fatalf("%s should not contain retired Python root-script execution guidance %q", relative, needle)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk repo surfaces: %v", err)
	}
}
