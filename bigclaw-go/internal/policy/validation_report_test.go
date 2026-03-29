package policy

import "testing"

func TestValidationReportPolicyBlocksIssueCloseWithoutRequiredReports(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay"})

	if decision.AllowedToClose || decision.Status != "blocked" {
		t.Fatalf("expected blocked decision, got %+v", decision)
	}
	if len(decision.MissingReports) != 1 || decision.MissingReports[0] != "benchmark-suite" {
		t.Fatalf("expected benchmark-suite as missing report, got %+v", decision.MissingReports)
	}
}

func TestValidationReportPolicyAllowsIssueCloseWhenReportsComplete(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay", "benchmark-suite"})

	if !decision.AllowedToClose || decision.Status != "ready" {
		t.Fatalf("expected ready decision, got %+v", decision)
	}
	if len(decision.MissingReports) != 0 {
		t.Fatalf("expected no missing reports, got %+v", decision.MissingReports)
	}
}
