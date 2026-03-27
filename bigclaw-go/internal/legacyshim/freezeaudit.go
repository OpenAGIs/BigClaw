package legacyshim

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type FreezeAuditResult struct {
	RootDir            string   `json:"root_dir"`
	Readme             string   `json:"readme"`
	Files              []string `json:"files"`
	CheckedEntrypoints []string `json:"checked_entrypoints"`
	MissingMarkers     []string `json:"missing_markers,omitempty"`
	MissingFiles       []string `json:"missing_files,omitempty"`
}

func FrozenEntrypointFiles(repoRoot string) []string {
	relative := []string{
		"src/bigclaw/__init__.py",
		"src/bigclaw/__main__.py",
		"src/bigclaw/service.py",
		"src/bigclaw/legacy_shim.py",
		"src/bigclaw/runtime.py",
		"src/bigclaw/scheduler.py",
		"src/bigclaw/workflow.py",
		"src/bigclaw/queue.py",
		"src/bigclaw/orchestration.py",
	}
	files := make([]string, 0, len(relative))
	for _, item := range relative {
		files = append(files, filepath.Join(repoRoot, item))
	}
	return files
}

func FreezeAudit(repoRoot string) (FreezeAuditResult, error) {
	rootDir := filepath.Join(repoRoot, "src/bigclaw")
	result := FreezeAuditResult{
		RootDir:            rootDir,
		Readme:             filepath.Join(rootDir, "README.md"),
		CheckedEntrypoints: FrozenEntrypointFiles(repoRoot),
	}
	files, err := filepath.Glob(filepath.Join(rootDir, "*.py"))
	if err != nil {
		return result, err
	}
	sort.Strings(files)
	result.Files = files
	if _, err := os.Stat(result.Readme); err != nil {
		if os.IsNotExist(err) {
			result.MissingFiles = append(result.MissingFiles, result.Readme)
			return result, nil
		}
		return result, err
	}
	for _, file := range result.CheckedEntrypoints {
		body, err := os.ReadFile(file)
		if err != nil {
			if os.IsNotExist(err) {
				result.MissingFiles = append(result.MissingFiles, file)
				continue
			}
			return result, err
		}
		if !hasFreezeMarker(body) {
			result.MissingMarkers = append(result.MissingMarkers, file)
		}
	}
	sort.Strings(result.MissingFiles)
	sort.Strings(result.MissingMarkers)
	return result, nil
}

func hasFreezeMarker(body []byte) bool {
	head := strings.ToLower(string(body))
	if len(head) > 800 {
		head = head[:800]
	}
	return strings.Contains(head, "frozen") ||
		strings.Contains(head, "migration-only") ||
		strings.Contains(head, "compatibility shim")
}
