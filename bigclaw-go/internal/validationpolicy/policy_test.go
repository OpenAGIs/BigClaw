package validationpolicy

import (
	"reflect"
	"testing"
)

func TestValidationPolicyBlocksIssueCloseWithoutRequiredReports(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay"})

	if decision.AllowedToClose || decision.Status != "blocked" {
		t.Fatalf("expected blocked decision, got %+v", decision)
	}
	if !reflect.DeepEqual(decision.MissingReports, []string{"benchmark-suite"}) {
		t.Fatalf("unexpected missing reports: %+v", decision.MissingReports)
	}
}

func TestValidationPolicyAllowsIssueCloseWhenReportsComplete(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay", "benchmark-suite"})

	if !decision.AllowedToClose || decision.Status != "ready" {
		t.Fatalf("expected ready decision, got %+v", decision)
	}
	if len(decision.MissingReports) != 0 {
		t.Fatalf("expected no missing reports, got %+v", decision.MissingReports)
	}
}
