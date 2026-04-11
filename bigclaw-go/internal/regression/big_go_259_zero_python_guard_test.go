package regression

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestBIGGO259RepositoryHasNoPythonLikeFiles(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	pythonFiles := collectPythonLikeFiles(t, rootRepo)
	if len(pythonFiles) != 0 {
		t.Fatalf("expected repository to remain Python-free, found %d Python-like file(s): %v", len(pythonFiles), pythonFiles)
	}

	pythonScripts := collectPythonShebangScripts(t, rootRepo)
	if len(pythonScripts) != 0 {
		t.Fatalf("expected repository to remain free of executable Python shebang scripts, found %d file(s): %v", len(pythonScripts), pythonScripts)
	}
}

func TestBIGGO259AuxiliaryResidualDirectoriesStayPythonFree(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	auxiliaryDirs := []string{
		".github",
		".githooks",
		".symphony",
		"docs/reports",
		"reports",
		"scripts/ops",
		"bigclaw-go/docs/reports",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts",
		"bigclaw-go/docs/reports/live-shadow-runs",
		"bigclaw-go/docs/reports/live-validation-runs",
	}

	for _, relativeDir := range auxiliaryDirs {
		root := filepath.Join(rootRepo, filepath.FromSlash(relativeDir))
		pythonFiles := collectPythonLikeFiles(t, root)
		if len(pythonFiles) != 0 {
			t.Fatalf("expected auxiliary residual directory to remain Python-free: %s (%v)", relativeDir, pythonFiles)
		}
		pythonScripts := collectPythonShebangScripts(t, root)
		if len(pythonScripts) != 0 {
			t.Fatalf("expected auxiliary residual directory to remain free of executable Python shebang scripts: %s (%v)", relativeDir, pythonScripts)
		}
	}
}

func TestBIGGO259RetainedNativeEvidencePathsRemainAvailable(t *testing.T) {
	rootRepo := regressionRepoRoot(t)

	retainedPaths := []string{
		".github/workflows/ci.yml",
		".githooks/post-commit",
		".symphony/workpad.md",
		"reports/BIG-GO-223-validation.md",
		"docs/reports/bootstrap-cache-validation.md",
		"scripts/ops/bigclawctl",
		"scripts/ops/bigclaw-issue",
		"scripts/ops/bigclaw-panel",
		"scripts/ops/bigclaw-symphony",
		"bigclaw-go/docs/reports/live-shadow-index.md",
		"bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/fault-timeline.json",
		"bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/idle-primary-takeover-live/node-b-audit.jsonl",
		"bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json",
		"bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json",
	}

	for _, relativePath := range retainedPaths {
		if _, err := os.Stat(filepath.Join(rootRepo, filepath.FromSlash(relativePath))); err != nil {
			t.Fatalf("expected retained native evidence path to exist: %s (%v)", relativePath, err)
		}
	}
}

func TestBIGGO259LaneReportCapturesSweepState(t *testing.T) {
	rootRepo := regressionRepoRoot(t)
	report := readRepoFile(t, rootRepo, "bigclaw-go/docs/reports/big-go-259-python-asset-sweep.md")

	for _, needle := range []string{
		"BIG-GO-259",
		"Repository-wide Python-like file count: `0`.",
		"Repository-wide executable Python shebang count: `0`.",
		"`.github`: `0` Python-like files, `0` executable Python shebang scripts",
		"`.githooks`: `0` Python-like files, `0` executable Python shebang scripts",
		"`.symphony`: `0` Python-like files, `0` executable Python shebang scripts",
		"`docs/reports`: `0` Python-like files, `0` executable Python shebang scripts",
		"`reports`: `0` Python-like files, `0` executable Python shebang scripts",
		"`scripts/ops`: `0` Python-like files, `0` executable Python shebang scripts",
		"`bigclaw-go/docs/reports`: `0` Python-like files, `0` executable Python shebang scripts",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts`: `0` Python-like files, `0` executable Python shebang scripts",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts`: `0` Python-like files, `0` executable Python shebang scripts",
		"`bigclaw-go/docs/reports/live-shadow-runs`: `0` Python-like files, `0` executable Python shebang scripts",
		"`bigclaw-go/docs/reports/live-validation-runs`: `0` Python-like files, `0` executable Python shebang scripts",
		"`.github/workflows/ci.yml`",
		"`.githooks/post-commit`",
		"`.symphony/workpad.md`",
		"`reports/BIG-GO-223-validation.md`",
		"`docs/reports/bootstrap-cache-validation.md`",
		"`scripts/ops/bigclawctl`",
		"`scripts/ops/bigclaw-issue`",
		"`scripts/ops/bigclaw-panel`",
		"`scripts/ops/bigclaw-symphony`",
		"`bigclaw-go/docs/reports/live-shadow-index.md`",
		"`bigclaw-go/docs/reports/broker-failover-stub-artifacts/BF-08/fault-timeline.json`",
		"`bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts/idle-primary-takeover-live/node-b-audit.jsonl`",
		"`bigclaw-go/docs/reports/live-shadow-runs/20260313T085655Z/summary.json`",
		"`bigclaw-go/docs/reports/live-validation-runs/20260316T140138Z/summary.json`",
		"`find . -path '*/.git' -prune -o -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) -print | sort`",
		"`find .github .githooks .symphony docs/reports reports scripts/ops bigclaw-go/docs/reports bigclaw-go/docs/reports/broker-failover-stub-artifacts bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-artifacts bigclaw-go/docs/reports/live-shadow-runs bigclaw-go/docs/reports/live-validation-runs -type f \\( -name '*.py' -o -name '*.pyw' -o -name '*.pyi' -o -name '*.ipynb' \\) 2>/dev/null | sort`",
		"`find . -path '*/.git' -prune -o -type f -perm -u+x -print | while IFS= read -r f; do first=$(LC_ALL=C sed -n '1p' \"$f\" 2>/dev/null || true); case \"$first\" in '#!'*python*) printf '%s\\n' \"$f\";; esac; done | sort`",
		"`cd bigclaw-go && go test -count=1 ./internal/regression -run 'TestBIGGO259(RepositoryHasNoPythonLikeFiles|AuxiliaryResidualDirectoriesStayPythonFree|RetainedNativeEvidencePathsRemainAvailable|LaneReportCapturesSweepState)$'`",
	} {
		if !strings.Contains(report, needle) {
			t.Fatalf("lane report missing substring %q", needle)
		}
	}
}

func collectPythonLikeFiles(t *testing.T, root string) []string {
	t.Helper()

	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if !hasPythonLikeExtension(path) {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(entries)
	return entries
}

func collectPythonShebangScripts(t *testing.T, root string) []string {
	t.Helper()

	entries := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return statErr
		}
		if info.Mode().Perm()&0o111 == 0 {
			return nil
		}
		firstLine, readErr := readFirstLine(path)
		if readErr != nil {
			return readErr
		}
		if !strings.HasPrefix(firstLine, "#!") || !strings.Contains(strings.ToLower(firstLine), "python") {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		entries = append(entries, filepath.ToSlash(relative))
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk %s: %v", root, err)
	}

	sort.Strings(entries)
	return entries
}

func hasPythonLikeExtension(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".py", ".pyi", ".pyw", ".ipynb":
		return true
	default:
		return false
	}
}

func readFirstLine(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	line, err := bufio.NewReader(file).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
