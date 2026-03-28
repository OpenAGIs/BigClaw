package regression

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoOnlyRepoCutoverDocAndRemovedShimsStayAligned(t *testing.T) {
	root := repoRoot(t)
	doc := readRepoFile(t, root, "../docs/go-only-repo-cutover.md")

	for _, needle := range []string{
		"bigclaw-go/scripts/e2e/run_task_smoke.py",
		"bigclaw-go/scripts/benchmark/soak_local.py",
		"bigclaw-go/scripts/migration/shadow_compare.py",
		"go run ./cmd/bigclawctl automation e2e run-task-smoke",
		"go run ./cmd/bigclawctl automation benchmark soak-local",
		"go run ./cmd/bigclawctl automation migration shadow-compare",
		"bigclaw-go/scripts/e2e/export_validation_bundle.py",
		"scripts/ops/bigclawctl",
		"go test ./...",
	} {
		if !strings.Contains(doc, needle) {
			t.Fatalf("go-only cutover doc missing %q", needle)
		}
	}

	for _, removed := range []string{
		"scripts/e2e/run_task_smoke.py",
		"scripts/benchmark/soak_local.py",
		"scripts/migration/shadow_compare.py",
	} {
		if _, err := os.Stat(filepath.Join(root, removed)); !os.IsNotExist(err) {
			t.Fatalf("expected removed shim %s to stay deleted, err=%v", removed, err)
		}
	}
}
