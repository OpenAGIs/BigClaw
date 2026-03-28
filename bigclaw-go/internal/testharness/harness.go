package testharness

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// RepoRoot returns the bigclaw-go module root regardless of the calling package cwd.
func RepoRoot(tb testing.TB) string {
	tb.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		tb.Fatal("failed to resolve test harness location")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
}

// ProjectRoot returns the parent repository root that contains both bigclaw-go and legacy assets like src/ and tests/.
func ProjectRoot(tb testing.TB) string {
	tb.Helper()
	return filepath.Dir(RepoRoot(tb))
}

// LegacySrcRoot returns the project-level src directory used by the remaining Python assets.
func LegacySrcRoot(tb testing.TB) string {
	tb.Helper()
	return JoinProjectRoot(tb, "src")
}

func JoinRepoRoot(tb testing.TB, elems ...string) string {
	tb.Helper()
	parts := append([]string{RepoRoot(tb)}, elems...)
	return filepath.Join(parts...)
}

func JoinProjectRoot(tb testing.TB, elems ...string) string {
	tb.Helper()
	parts := append([]string{ProjectRoot(tb)}, elems...)
	return filepath.Join(parts...)
}

// ResolveProjectPath maps repo-relative paths that may still be prefixed with bigclaw-go/.
func ResolveProjectPath(tb testing.TB, candidate string) string {
	tb.Helper()
	return JoinRepoRoot(tb, strings.TrimPrefix(candidate, "bigclaw-go/"))
}

func PrependPathEnv(tb testing.TB, dir string) {
	tb.Helper()
	prependEnv(tb, "PATH", dir)
}

// PrependPythonPathEnv mirrors tests/conftest.py by prepending a directory to PYTHONPATH.
func PrependPythonPathEnv(tb testing.TB, dir string) {
	tb.Helper()
	prependEnv(tb, "PYTHONPATH", dir)
}

// BootstrapLegacyPythonPath prepends the legacy src/ root to PYTHONPATH and returns it.
func BootstrapLegacyPythonPath(tb testing.TB) string {
	tb.Helper()
	srcRoot := LegacySrcRoot(tb)
	PrependPythonPathEnv(tb, srcRoot)
	return srcRoot
}

func prependEnv(tb testing.TB, key, dir string) {
	tb.Helper()
	current := os.Getenv(key)
	if current == "" {
		tb.Setenv(key, dir)
		return
	}
	tb.Setenv(key, dir+string(os.PathListSeparator)+current)
}

func Chdir(tb testing.TB, dir string) {
	tb.Helper()
	original, err := os.Getwd()
	if err != nil {
		tb.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		tb.Fatalf("chdir %s: %v", dir, err)
	}
	tb.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			tb.Fatalf("restore cwd %s: %v", original, err)
		}
	})
}

type PytestAssetInventory struct {
	TestModules            []string
	BigclawImportModules   []string
	PytestImportModules    []string
	ConftestPath           string
	ConftestPrependsSrc    bool
	ConftestImportsPytest  bool
	ConftestDefinesFixture bool
	ConftestDefinesHook    bool
}

func InventoryPytestAssets(tb testing.TB) PytestAssetInventory {
	tb.Helper()

	testsDir := JoinProjectRoot(tb, "tests")
	entries, err := os.ReadDir(testsDir)
	if err != nil {
		tb.Fatalf("read pytest asset directory %s: %v", testsDir, err)
	}

	inventory := PytestAssetInventory{
		ConftestPath: filepath.Join(testsDir, "conftest.py"),
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".py" {
			continue
		}
		fullPath := filepath.Join(testsDir, name)
		relPath := filepath.ToSlash(filepath.Join("tests", name))
		if name == "conftest.py" {
			inventory.ConftestPrependsSrc = fileContains(tb, fullPath, `sys.path.insert(0, str(SRC))`)
			inventory.ConftestImportsPytest = fileContains(tb, fullPath, "import pytest") || fileContains(tb, fullPath, "from pytest")
			inventory.ConftestDefinesFixture = fileContains(tb, fullPath, "@pytest.fixture") || fileContains(tb, fullPath, "def fixture_")
			inventory.ConftestDefinesHook = fileContains(tb, fullPath, "def pytest_")
			continue
		}
		inventory.TestModules = append(inventory.TestModules, relPath)
		if fileContains(tb, fullPath, "from bigclaw") || fileContains(tb, fullPath, "import bigclaw") {
			inventory.BigclawImportModules = append(inventory.BigclawImportModules, relPath)
		}
		if fileContains(tb, fullPath, "import pytest") || fileContains(tb, fullPath, "pytest.") {
			inventory.PytestImportModules = append(inventory.PytestImportModules, relPath)
		}
	}

	slices.Sort(inventory.TestModules)
	slices.Sort(inventory.BigclawImportModules)
	slices.Sort(inventory.PytestImportModules)
	return inventory
}

func fileContains(tb testing.TB, path, needle string) bool {
	tb.Helper()
	file, err := os.Open(path)
	if err != nil {
		tb.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), needle) {
			return true
		}
	}
	if err := scanner.Err(); err != nil {
		tb.Fatalf("scan %s: %v", path, err)
	}
	return false
}

func (i PytestAssetInventory) Summary() string {
	return "tests=" + strconv.Itoa(len(i.TestModules)) +
		" bigclaw_imports=" + strconv.Itoa(len(i.BigclawImportModules)) +
		" pytest_imports=" + strconv.Itoa(len(i.PytestImportModules))
}
