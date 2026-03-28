package testharness

import (
	"os"
	"path/filepath"
	"runtime"
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
	current := os.Getenv("PATH")
	if current == "" {
		tb.Setenv("PATH", dir)
		return
	}
	tb.Setenv("PATH", dir+string(os.PathListSeparator)+current)
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
