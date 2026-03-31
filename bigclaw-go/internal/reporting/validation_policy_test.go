package reporting

import "testing"

func TestValidationPolicyBlocksIssueCloseWithoutRequiredReports(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay"})

	if decision.AllowedToClose {
		t.Fatalf("expected close to be blocked, got %+v", decision)
	}
	if decision.Status != "blocked" {
		t.Fatalf("expected blocked status, got %+v", decision)
	}
	if len(decision.MissingReports) != 1 || decision.MissingReports[0] != "benchmark-suite" {
		t.Fatalf("expected missing benchmark-suite, got %+v", decision)
	}
}

func TestValidationPolicyAllowsIssueCloseWhenReportsComplete(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay", "benchmark-suite"})

	if !decision.AllowedToClose {
		t.Fatalf("expected close to be allowed, got %+v", decision)
	}
	if decision.Status != "ready" {
		t.Fatalf("expected ready status, got %+v", decision)
	}
}
