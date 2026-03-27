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
		"docs/reports/go-migration-plan-summary.json",
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

func TestIssueCoverageReferencesGoMigrationInventory(t *testing.T) {
	repoRoot := repoRoot(t)
	contents := readRepoFile(t, repoRoot, "docs/reports/issue-coverage.md")

	requiredSubstrings := []string{
		"`OPE-185` / `BIG-GO-010`",
		"docs/go-migration-inventory.md",
		"docs/reports/go-migration-ledger.json",
		"docs/reports/go-migration-plan-summary.json",
		"`BIG-GO-901` adds the 100% non-Go executable asset inventory",
	}
	for _, needle := range requiredSubstrings {
		if !strings.Contains(contents, needle) {
			t.Fatalf("docs/reports/issue-coverage.md missing substring %q", needle)
		}
	}
}

func TestGoMigrationPlanSummaryStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	ledger := readGoMigrationLedger(t, repoRoot)

	var summary struct {
		Issue struct {
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
		} `json:"issue"`
		InventorySummary struct {
			TotalAssets   int            `json:"total_assets"`
			PythonModules int            `json:"python_modules"`
			PythonScripts int            `json:"python_scripts"`
			ShellWrappers int            `json:"shell_wrappers"`
			Priorities    map[string]int `json:"priorities"`
		} `json:"inventory_summary"`
		Waves          []map[string]any `json:"waves"`
		FirstWaveLanes []struct {
			Name   string   `json:"name"`
			Branch string   `json:"branch"`
			Assets []string `json:"assets"`
		} `json:"first_wave_lanes"`
		ValidationSuites []struct {
			Name    string `json:"name"`
			Command string `json:"command"`
		} `json:"validation_suites"`
		MajorRisks []struct {
			Risk       string `json:"risk"`
			Mitigation string `json:"mitigation"`
		} `json:"major_risks"`
	}

	body := readRepoFile(t, repoRoot, "docs/reports/go-migration-plan-summary.json")
	if err := json.Unmarshal([]byte(body), &summary); err != nil {
		t.Fatalf("unmarshal go migration plan summary: %v", err)
	}

	if summary.Issue.Identifier != "BIG-GO-901" || summary.Issue.Title != "Go迁移盘点与目标架构" {
		t.Fatalf("unexpected issue metadata: %+v", summary.Issue)
	}
	if summary.InventorySummary.TotalAssets != ledger.Summary.TotalAssets ||
		summary.InventorySummary.PythonModules != ledger.Summary.PythonModules ||
		summary.InventorySummary.PythonScripts != ledger.Summary.PythonScripts ||
		summary.InventorySummary.ShellWrappers != ledger.Summary.ShellWrappers {
		t.Fatalf("plan summary inventory drifted from ledger: %+v vs %+v", summary.InventorySummary, ledger.Summary)
	}
	if len(summary.Waves) != 4 {
		t.Fatalf("expected 4 migration waves, got %d", len(summary.Waves))
	}
	if len(summary.FirstWaveLanes) != 5 {
		t.Fatalf("expected 5 first-wave lanes, got %d", len(summary.FirstWaveLanes))
	}
	if len(summary.ValidationSuites) != 6 {
		t.Fatalf("expected 6 validation suites, got %d", len(summary.ValidationSuites))
	}
	if len(summary.MajorRisks) != 5 {
		t.Fatalf("expected 5 major risks, got %d", len(summary.MajorRisks))
	}

	expectedLanes := map[string]string{
		"lane/core-cutover":      "codex/big-go-901-core-cutover",
		"lane/contracts-runtime": "codex/big-go-901-contracts-runtime",
		"lane/migration-cli":     "codex/big-go-901-migration-cli",
		"lane/validation-cli":    "codex/big-go-901-validation-cli",
		"lane/control-surface":   "codex/big-go-901-control-surface",
	}
	for _, lane := range summary.FirstWaveLanes {
		expectedBranch, ok := expectedLanes[lane.Name]
		if !ok {
			t.Fatalf("unexpected lane %q", lane.Name)
		}
		if lane.Branch != expectedBranch {
			t.Fatalf("lane %q branch mismatch: got %q want %q", lane.Name, lane.Branch, expectedBranch)
		}
		if len(lane.Assets) == 0 {
			t.Fatalf("lane %q must list assets", lane.Name)
		}
	}

	for _, suite := range summary.ValidationSuites {
		if !strings.Contains(suite.Command, "go test") {
			t.Fatalf("validation suite %q missing go test command: %q", suite.Name, suite.Command)
		}
	}
	for _, risk := range summary.MajorRisks {
		if strings.TrimSpace(risk.Risk) == "" || strings.TrimSpace(risk.Mitigation) == "" {
			t.Fatalf("risk entry missing content: %+v", risk)
		}
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
