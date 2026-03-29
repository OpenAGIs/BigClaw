package validationpolicy

import (
	"reflect"
	"testing"
)

func TestValidationPolicyBlocksIssueCloseWithoutRequiredReports(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay"})

	if decision.AllowedToClose {
		t.Fatalf("expected close to be blocked, got %+v", decision)
	}
	if decision.Status != "blocked" {
		t.Fatalf("expected blocked status, got %+v", decision)
	}
	if !reflect.DeepEqual(decision.MissingReports, []string{"benchmark-suite"}) {
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
	if len(decision.MissingReports) != 0 {
		t.Fatalf("expected no missing reports, got %+v", decision)
	}
}
