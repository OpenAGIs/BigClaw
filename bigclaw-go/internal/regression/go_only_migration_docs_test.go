package regression

import (
	"strings"
	"testing"
)

func TestGoOnlyMigrationPlanDocsStayAligned(t *testing.T) {
	root := repoRoot(t)
	plan := readRepoFile(t, root, "../docs/go-only-migration-plan.md")
	for _, needle := range []string{
		"docs/reports/go-only-migration-inventory.json",
		"BIG-VNEXT-GO-101",
		"BIG-VNEXT-GO-110",
		"## Branch And PR Strategy",
		"Start `BIG-VNEXT-GO-102`, `BIG-VNEXT-GO-103`, and `BIG-VNEXT-GO-104` in parallel",
	} {
		if !strings.Contains(plan, needle) {
			t.Fatalf("migration plan missing %q", needle)
		}
	}

	inventory := readRepoFile(t, root, "../docs/reports/go-only-migration-inventory.json")
	for _, needle := range []string{
		"\"parallel_slice_count\": 10",
		"\"identifier\": \"BIG-VNEXT-GO-101\"",
		"\"identifier\": \"BIG-VNEXT-GO-110\"",
		"\"branch_prefix\": \"symphony/\"",
	} {
		if !strings.Contains(inventory, needle) {
			t.Fatalf("migration inventory missing %q", needle)
		}
	}
}
