package regression

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type goMigrationLedger struct {
	Summary struct {
		TotalAssets   int            `json:"total_assets"`
		PythonModules int            `json:"python_modules"`
		PythonScripts int            `json:"python_scripts"`
		ShellWrappers int            `json:"shell_wrappers"`
		Priorities    map[string]int `json:"priorities"`
	} `json:"summary"`
	Entries []goMigrationLedgerEntry `json:"entries"`
}

type goMigrationLedgerEntry struct {
	Asset      string `json:"asset"`
	Kind       string `json:"kind"`
	Priority   string `json:"priority"`
	Target     string `json:"target"`
	Action     string `json:"action"`
	Validation string `json:"validation"`
	Regression string `json:"regression"`
}

func TestGoMigrationLedgerCoversCurrentNonGoAssets(t *testing.T) {
	repoRoot := repoRoot(t)
	workspaceRoot := filepath.Clean(filepath.Join(repoRoot, ".."))
	ledger := readGoMigrationLedger(t, repoRoot)

	expected := collectGoMigrationAssets(t, workspaceRoot)
	actual := make([]string, 0, len(ledger.Entries))
	seen := make(map[string]struct{}, len(ledger.Entries))
	for _, entry := range ledger.Entries {
		if strings.TrimSpace(entry.Asset) == "" {
			t.Fatal("ledger entry has empty asset path")
		}
		if _, ok := seen[entry.Asset]; ok {
			t.Fatalf("duplicate ledger asset %q", entry.Asset)
		}
		seen[entry.Asset] = struct{}{}
		if strings.TrimSpace(entry.Kind) == "" || strings.TrimSpace(entry.Priority) == "" {
			t.Fatalf("ledger entry %q missing required metadata", entry.Asset)
		}
		if strings.TrimSpace(entry.Target) == "" || strings.TrimSpace(entry.Action) == "" {
			t.Fatalf("ledger entry %q missing target/action", entry.Asset)
		}
		if strings.TrimSpace(entry.Validation) == "" || strings.TrimSpace(entry.Regression) == "" {
			t.Fatalf("ledger entry %q missing validation/regression", entry.Asset)
		}
		actual = append(actual, entry.Asset)
	}

	slices.Sort(expected)
	slices.Sort(actual)
	if !slices.Equal(expected, actual) {
		t.Fatalf("ledger assets mismatch\nexpected=%v\nactual=%v", expected, actual)
	}

	modules, scripts, wrappers := countLedgerKinds(ledger.Entries)
	if ledger.Summary.TotalAssets != len(expected) {
		t.Fatalf("unexpected total_assets: got %d want %d", ledger.Summary.TotalAssets, len(expected))
	}
	if ledger.Summary.PythonModules != modules || ledger.Summary.PythonScripts != scripts || ledger.Summary.ShellWrappers != wrappers {
		t.Fatalf("unexpected kind summary: %+v modules=%d scripts=%d wrappers=%d", ledger.Summary, modules, scripts, wrappers)
	}
	if sumPriorities(ledger.Summary.Priorities) != len(expected) {
		t.Fatalf("priority summary does not add up: %+v", ledger.Summary.Priorities)
	}
}

func TestGoMigrationInventoryDocKeepsRequiredSections(t *testing.T) {
	repoRoot := repoRoot(t)
	doc := readRepoFile(t, repoRoot, "docs/go-migration-inventory.md")
	ledger := readGoMigrationLedger(t, repoRoot)

	requiredSections := []string{
		"## 当前资产总览",
		"## 目标架构",
		"## 可执行迁移方案",
		"## 首批实现/改造清单",
		"## 验证命令与回归面",
		"## 分支与 PR 建议",
		"## 主要风险",
	}
	for _, section := range requiredSections {
		if !strings.Contains(doc, section) {
			t.Fatalf("migration inventory doc missing %q", section)
		}
	}

	requiredCounts := []string{
		"76",
		"49",
		"23",
		"4",
		"22",
		"44",
		"10",
	}
	for _, count := range requiredCounts {
		if !strings.Contains(doc, count) {
			t.Fatalf("migration inventory doc missing count token %q", count)
		}
	}

	if ledger.Summary.TotalAssets != 76 || ledger.Summary.PythonModules != 49 || ledger.Summary.PythonScripts != 23 || ledger.Summary.ShellWrappers != 4 {
		t.Fatalf("guarded summary changed; update this test with the new accepted inventory totals: %+v", ledger.Summary)
	}
}

func readGoMigrationLedger(t *testing.T, repoRoot string) goMigrationLedger {
	t.Helper()
	body := readRepoFile(t, repoRoot, "docs/reports/go-migration-ledger.json")
	var ledger goMigrationLedger
	if err := json.Unmarshal([]byte(body), &ledger); err != nil {
		t.Fatalf("unmarshal go migration ledger: %v", err)
	}
	return ledger
}

func collectGoMigrationAssets(t *testing.T, workspaceRoot string) []string {
	t.Helper()
	var assets []string

	pythonModules, err := filepath.Glob(filepath.Join(workspaceRoot, "src", "bigclaw", "*.py"))
	if err != nil {
		t.Fatalf("glob python modules: %v", err)
	}
	for _, path := range pythonModules {
		if filepath.Base(path) == "__init__.py" {
			continue
		}
		relative, err := filepath.Rel(workspaceRoot, path)
		if err != nil {
			t.Fatalf("rel python module %s: %v", path, err)
		}
		assets = append(assets, filepath.ToSlash(relative))
	}

	scriptRoot := filepath.Join(workspaceRoot, "bigclaw-go", "scripts")
	if err := filepath.WalkDir(scriptRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ext := filepath.Ext(path); ext != ".py" && ext != ".sh" {
			return nil
		}
		relative, relErr := filepath.Rel(workspaceRoot, path)
		if relErr != nil {
			return relErr
		}
		assets = append(assets, filepath.ToSlash(relative))
		return nil
	}); err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("scripts root missing: %v", err)
		}
		t.Fatalf("walk scripts root: %v", err)
	}

	if len(assets) == 0 {
		t.Fatal("expected non-Go assets for migration inventory, got none")
	}
	return assets
}

func countLedgerKinds(entries []goMigrationLedgerEntry) (modules, scripts, wrappers int) {
	for _, entry := range entries {
		switch entry.Kind {
		case "python-module":
			modules++
		case "python-script":
			scripts++
		case "shell-wrapper":
			wrappers++
		}
	}
	return modules, scripts, wrappers
}

func sumPriorities(priorities map[string]int) int {
	total := 0
	for _, count := range priorities {
		total += count
	}
	return total
}
