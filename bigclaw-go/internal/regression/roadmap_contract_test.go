package regression

import (
	"fmt"
	"strings"
	"testing"
)

type executionPackEpic struct {
	EpicID    string
	Phase     string
	Owner     string
	Milestone string
}

func TestExecutionPackRoadmapDocsStayAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	issuePlan := readRepoFile(t, repoRoot, "../docs/issue-plan.md")
	syncSummary := readRepoFile(t, repoRoot, "docs/reports/linear-project-sync-summary.md")

	requiredIssuePlan := []string{
		"BIG-EPIC-8: 研发自治执行平台增强",
		"BIG-EPIC-9: 工程运营系统",
		"BIG-EPIC-10: 跨部门 Agent Orchestration",
		"BIG-EPIC-11: 产品化 UI 与控制台",
		"BIG-EPIC-12: 计费、套餐与商业化控制",
		"BIG-EPIC-8: Phase 1 / M1 Foundation uplift / owner `engineering-platform`",
		"BIG-EPIC-9: Phase 2 / M2 Operations control plane / owner `engineering-operations`",
		"BIG-EPIC-10: Phase 3 / M3 Cross-team orchestration / owner `orchestration-office`",
		"BIG-EPIC-11: Phase 4 / M4 Productized console / owner `product-experience`",
		"BIG-EPIC-12: Phase 5 / M5 Billing and packaging / owner `commercialization`",
	}
	for _, needle := range requiredIssuePlan {
		if !strings.Contains(issuePlan, needle) {
			t.Fatalf("docs/issue-plan.md missing substring %q", needle)
		}
	}

	if !strings.Contains(syncSummary, "## BigClaw v4.0 Execution Pack") {
		t.Fatalf("linear project sync summary missing execution pack heading")
	}
}

func TestExecutionPackRoadmapUniqueOwnersContract(t *testing.T) {
	epics := []executionPackEpic{
		{EpicID: "BIG-EPIC-8", Phase: "Phase 1", Owner: "engineering-platform", Milestone: "M1 Foundation uplift"},
		{EpicID: "BIG-EPIC-9", Phase: "Phase 2", Owner: "engineering-operations", Milestone: "M2 Operations control plane"},
		{EpicID: "BIG-EPIC-10", Phase: "Phase 3", Owner: "orchestration-office", Milestone: "M3 Cross-team orchestration"},
		{EpicID: "BIG-EPIC-11", Phase: "Phase 4", Owner: "product-experience", Milestone: "M4 Productized console"},
		{EpicID: "BIG-EPIC-12", Phase: "Phase 5", Owner: "commercialization", Milestone: "M5 Billing and packaging"},
	}
	if err := validateUniqueExecutionPackOwners(epics); err != nil {
		t.Fatalf("expected canonical roadmap owners to be unique, got %v", err)
	}
	if epics[2].Milestone != "M3 Cross-team orchestration" {
		t.Fatalf("unexpected phase 3 milestone: %+v", epics[2])
	}

	err := validateUniqueExecutionPackOwners([]executionPackEpic{
		{EpicID: "BIG-EPIC-8", Owner: "engineering-platform"},
		{EpicID: "BIG-EPIC-9", Owner: "engineering-platform"},
	})
	if err == nil || err.Error() != "Owner 'engineering-platform' is assigned to both BIG-EPIC-8 and BIG-EPIC-9" {
		t.Fatalf("unexpected duplicate owner validation error: %v", err)
	}
}

func validateUniqueExecutionPackOwners(epics []executionPackEpic) error {
	seen := map[string]string{}
	for _, epic := range epics {
		ownerKey := strings.ToLower(strings.TrimSpace(epic.Owner))
		if ownerKey == "" {
			continue
		}
		if firstEpic, ok := seen[ownerKey]; ok {
			return fmt.Errorf("Owner '%s' is assigned to both %s and %s", epic.Owner, firstEpic, epic.EpicID)
		}
		seen[ownerKey] = epic.EpicID
	}
	return nil
}
