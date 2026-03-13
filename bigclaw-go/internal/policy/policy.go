package policy

import (
	"fmt"
	"strings"

	"bigclaw-go/internal/domain"
)

type Summary struct {
	Plan                 string `json:"plan"`
	DedicatedQueue       string `json:"dedicated_queue"`
	ConcurrencyProfile   string `json:"concurrency_profile"`
	AdvancedApproval     bool   `json:"advanced_approval"`
	MultiAgentGraph      bool   `json:"multi_agent_graph"`
	DedicatedBrowserPool bool   `json:"dedicated_browser_pool"`
	DedicatedVMPool      bool   `json:"dedicated_vm_pool"`
	Isolation            string `json:"isolation"`
	Reason               string `json:"reason"`
}

func Resolve(task domain.Task) Summary {
	premium := false
	reason := "default shared orchestration"
	switch {
	case metadataEquals(task, "plan", "premium"), metadataEquals(task, "tier", "premium"), metadataEquals(task, "orchestration", "premium"), hasLabel(task, "premium"):
		premium = true
		reason = "metadata requested premium orchestration"
	case task.RiskLevel == domain.RiskHigh && (requiresTool(task, "browser") || requiresTool(task, "gpu") || requiresTool(task, "vm")):
		premium = true
		reason = "high-risk tool workload promoted to premium lane"
	}

	if premium {
		return Summary{
			Plan:                 "premium",
			DedicatedQueue:       queueName(task, "premium"),
			ConcurrencyProfile:   "elevated",
			AdvancedApproval:     true,
			MultiAgentGraph:      true,
			DedicatedBrowserPool: requiresTool(task, "browser"),
			DedicatedVMPool:      requiresTool(task, "vm") || task.RequiredExecutor == domain.ExecutorKubernetes || task.RequiredExecutor == domain.ExecutorRay,
			Isolation:            "dedicated",
			Reason:               reason,
		}
	}

	return Summary{
		Plan:               "standard",
		DedicatedQueue:     queueName(task, "shared"),
		ConcurrencyProfile: "shared",
		Isolation:          "shared",
		Reason:             reason,
	}
}

func queueName(task domain.Task, lane string) string {
	team := strings.TrimSpace(task.Metadata["team"])
	if team == "" {
		team = strings.TrimSpace(task.TenantID)
	}
	if team == "" {
		return lane + "/default"
	}
	return fmt.Sprintf("%s/%s", lane, team)
}

func metadataEquals(task domain.Task, key, want string) bool {
	return strings.EqualFold(strings.TrimSpace(task.Metadata[key]), want)
}

func requiresTool(task domain.Task, tool string) bool {
	for _, item := range task.RequiredTools {
		if strings.EqualFold(item, tool) {
			return true
		}
	}
	return false
}

func hasLabel(task domain.Task, label string) bool {
	for _, item := range task.Labels {
		if strings.EqualFold(item, label) {
			return true
		}
	}
	return false
}
