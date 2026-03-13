package policy

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestResolvePremiumPolicyFromMetadata(t *testing.T) {
	summary := Resolve(domain.Task{
		TenantID:      "platform",
		RequiredTools: []string{"browser", "vm"},
		Metadata:      map[string]string{"plan": "premium", "team": "platform"},
	})
	if summary.Plan != "premium" || !summary.AdvancedApproval || !summary.MultiAgentGraph {
		t.Fatalf("expected premium policy, got %+v", summary)
	}
	if summary.DedicatedQueue != "premium/platform" {
		t.Fatalf("unexpected dedicated queue, got %+v", summary)
	}
}
