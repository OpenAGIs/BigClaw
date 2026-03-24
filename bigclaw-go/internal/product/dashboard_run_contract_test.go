package product

import (
	"strings"
	"testing"
)

func TestBuildDefaultDashboardRunContractIsReleaseReady(t *testing.T) {
	contract := BuildDefaultDashboardRunContract()
	audit := AuditDashboardRunContract(contract)
	if !audit.ReleaseReady {
		t.Fatalf("expected default contract to be release ready, got %+v", audit)
	}
	if contract.ContractID != "BIG-GOM-305" || contract.Version != "go-v1" {
		t.Fatalf("unexpected contract metadata: %+v", contract)
	}
	if len(contract.DashboardSchema.Fields) == 0 || len(contract.RunDetailSchema.Fields) == 0 {
		t.Fatalf("expected populated schema fields, got %+v", contract)
	}
}

func TestDashboardRunContractAuditDetectsMissingPaths(t *testing.T) {
	contract := BuildDefaultDashboardRunContract()
	contract.DashboardSchema.Fields = contract.DashboardSchema.Fields[:len(contract.DashboardSchema.Fields)-1]
	delete(contract.RunDetailSchema.Sample["closeout"].(map[string]any), "git_log_stat_output")

	audit := AuditDashboardRunContract(contract)
	if audit.ReleaseReady {
		t.Fatalf("expected audit to detect gaps, got %+v", audit)
	}
	if len(audit.DashboardMissingFields) == 0 || len(audit.RunDetailSampleGaps) == 0 {
		t.Fatalf("expected missing field and sample gap findings, got %+v", audit)
	}
}

func TestRenderDashboardRunContractReport(t *testing.T) {
	contract := BuildDefaultDashboardRunContract()
	audit := AuditDashboardRunContract(contract)
	report := RenderDashboardRunContractReport(contract, audit)
	for _, want := range []string{"# Dashboard and Run Contract", "engineering-dashboard-platform-alpha", "\"closeout\"", "Release Ready: true"} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
	}
}

func TestContractPathExistsHandlesNestedObjectsAndLists(t *testing.T) {
	payload := map[string]any{
		"closeout": map[string]any{
			"git_push_status": "ok",
			"artifacts": []map[string]any{
				{"path": "/tmp/one"},
			},
			"events": []any{
				map[string]any{"id": "evt-1", "status": "ok"},
			},
		},
	}

	for _, tc := range []struct {
		path   string
		expect bool
	}{
		{path: "closeout.git_push_status", expect: true},
		{path: "closeout.artifacts[].path", expect: true},
		{path: "closeout.events[].status", expect: true},
		{path: "closeout.events[].missing", expect: false},
		{path: "closeout.unknown", expect: false},
		{path: "closeout.git_push_status.value", expect: false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			if got := contractPathExists(payload, tc.path); got != tc.expect {
				t.Fatalf("expected contractPathExists(%q)=%t, got %t", tc.path, tc.expect, got)
			}
		})
	}
}
