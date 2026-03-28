package policy

import (
	"reflect"
	"testing"
)

func TestEnforceValidationReportPolicyBlocksCloseoutWithoutRequiredReports(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay"})

	if decision.AllowedToClose {
		t.Fatalf("expected closeout block, got %+v", decision)
	}
	if decision.Status != "blocked" || decision.Summary != "validation report policy not satisfied" {
		t.Fatalf("unexpected block decision: %+v", decision)
	}
	if !reflect.DeepEqual(decision.MissingReports, []string{"benchmark-suite"}) {
		t.Fatalf("unexpected missing reports: %+v", decision.MissingReports)
	}
}

func TestEnforceValidationReportPolicyAllowsCloseoutWhenReportsComplete(t *testing.T) {
	decision := EnforceValidationReportPolicy([]string{"task-run", "replay", "benchmark-suite"})

	if !decision.AllowedToClose {
		t.Fatalf("expected closeout ready decision, got %+v", decision)
	}
	if decision.Status != "ready" || decision.Summary != "validation report policy satisfied" {
		t.Fatalf("unexpected ready decision: %+v", decision)
	}
	if len(decision.MissingReports) != 0 {
		t.Fatalf("expected no missing reports, got %+v", decision.MissingReports)
	}
}
