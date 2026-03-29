package product

import (
	"encoding/json"
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
	delete(contract.DashboardSchema.Sample, "trend")
	delete(contract.RunDetailSchema.Sample["closeout"].(map[string]any), "git_log_stat_output")

	audit := AuditDashboardRunContract(contract)
	if audit.ReleaseReady {
		t.Fatalf("expected audit to detect gaps, got %+v", audit)
	}
	if len(audit.DashboardMissingFields) == 0 || len(audit.DashboardSampleGaps) == 0 || len(audit.RunDetailSampleGaps) == 0 {
		t.Fatalf("expected missing field and sample gap findings, got %+v", audit)
	}
	if audit.DashboardSampleGaps[0] != "trend" || audit.RunDetailSampleGaps[0] != "closeout.git_log_stat_output" {
		t.Fatalf("expected deterministic sample gap findings, got %+v", audit)
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

func TestContractPathExistsTraversesNestedObjectsAndLists(t *testing.T) {
	payload := map[string]any{
		"closeout": map[string]any{
			"commits": []map[string]any{
				{"sha": "abc123", "files": []any{"a.go", "b.go"}},
			},
			"checks": []any{
				map[string]any{"name": "unit", "status": "ok"},
			},
		},
	}

	if !contractPathExists(payload, "closeout.commits[].sha") {
		t.Fatal("expected path through []map[string]any to exist")
	}
	if !contractPathExists(payload, "closeout.checks[].status") {
		t.Fatal("expected path through []any list to exist")
	}
	if contractPathExists(payload, "closeout.commits[].author") {
		t.Fatal("expected missing nested key to report false")
	}
	if contractPathExists(payload, "closeout.missing[].status") {
		t.Fatal("expected missing list key to report false")
	}
}

func TestContractPathExistsSkipsNonMapListEntries(t *testing.T) {
	payload := map[string]any{
		"closeout": map[string]any{
			"checks": []any{
				"not-a-map",
				map[string]any{"status": "ok"},
			},
		},
	}

	if !contractPathExists(payload, "closeout.checks[].status") {
		t.Fatal("expected path walker to skip non-map entries and still find later object values")
	}
}

func TestDashboardContractFormattingHelpers(t *testing.T) {
	if got := fallbackJoin(nil); got != "none" {
		t.Fatalf("fallbackJoin(nil) = %q, want %q", got, "none")
	}
	if got := fallbackJoin([]string{"zulu", "alpha", "alpha"}); got != "alpha, alpha, zulu" {
		t.Fatalf("fallbackJoin sorts values = %q, want %q", got, "alpha, alpha, zulu")
	}
	if got := boolText(true); got != "true" {
		t.Fatalf("boolText(true) = %q, want %q", got, "true")
	}
	if got := boolText(false); got != "false" {
		t.Fatalf("boolText(false) = %q, want %q", got, "false")
	}
}

func TestDashboardRunContractJSONRoundTrip(t *testing.T) {
	contract := BuildDefaultDashboardRunContract()
	payload, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}

	var restored DashboardRunContract
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}

	if restored.ContractID != contract.ContractID || restored.Version != contract.Version {
		t.Fatalf("unexpected contract metadata after round trip: restored=%+v want=%+v", restored, contract)
	}
	if len(restored.DashboardSchema.Fields) != len(contract.DashboardSchema.Fields) || len(restored.RunDetailSchema.Fields) != len(contract.RunDetailSchema.Fields) {
		t.Fatalf("expected schema fields to survive round trip: restored=%+v want=%+v", restored, contract)
	}
	if restored.DashboardSchema.Fields[0].Name != "dashboard_id" || restored.RunDetailSchema.Sample["closeout"] == nil {
		t.Fatalf("expected representative fields and samples to survive round trip: restored=%+v", restored)
	}
}
