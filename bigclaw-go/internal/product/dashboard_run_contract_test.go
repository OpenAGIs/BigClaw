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

func TestDashboardRunContractAuditMatchesResidualPythonScenario(t *testing.T) {
	contract := BuildDefaultDashboardRunContract()
	filteredDashboardFields := make([]ContractField, 0, len(contract.DashboardSchema.Fields))
	for _, field := range contract.DashboardSchema.Fields {
		if field.Name != "summary.sla_risk_runs" {
			filteredDashboardFields = append(filteredDashboardFields, field)
		}
	}
	contract.DashboardSchema.Fields = filteredDashboardFields
	delete(contract.DashboardSchema.Sample, "trend")

	filteredRunDetailFields := make([]ContractField, 0, len(contract.RunDetailSchema.Fields))
	for _, field := range contract.RunDetailSchema.Fields {
		if field.Name != "closeout.git_log_stat_output" {
			filteredRunDetailFields = append(filteredRunDetailFields, field)
		}
	}
	contract.RunDetailSchema.Fields = filteredRunDetailFields
	delete(contract.RunDetailSchema.Sample["closeout"].(map[string]any), "git_log_stat_output")

	audit := AuditDashboardRunContract(contract)
	if got, want := audit.DashboardMissingFields, []string{"summary.sla_risk_runs"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected dashboard missing fields: got=%v want=%v", got, want)
	}
	if got, want := audit.DashboardSampleGaps, []string{"trend"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected dashboard sample gaps: got=%v want=%v", got, want)
	}
	if got, want := audit.RunDetailMissingFields, []string{"closeout.git_log_stat_output"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected run detail missing fields: got=%v want=%v", got, want)
	}
	if got, want := audit.RunDetailSampleGaps, []string{"closeout.git_log_stat_output"}; strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("unexpected run detail sample gaps: got=%v want=%v", got, want)
	}
	if audit.ReleaseReady {
		t.Fatalf("expected release readiness to fail, got %+v", audit)
	}
}

func TestDashboardRunContractJSONRoundTripPreservesContractAndAudit(t *testing.T) {
	contract := BuildDefaultDashboardRunContract()
	audit := AuditDashboardRunContract(contract)

	encodedContract, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("marshal contract: %v", err)
	}
	encodedAudit, err := json.Marshal(audit)
	if err != nil {
		t.Fatalf("marshal audit: %v", err)
	}

	var restoredContract DashboardRunContract
	if err := json.Unmarshal(encodedContract, &restoredContract); err != nil {
		t.Fatalf("unmarshal contract: %v", err)
	}
	var restoredAudit DashboardRunContractAudit
	if err := json.Unmarshal(encodedAudit, &restoredAudit); err != nil {
		t.Fatalf("unmarshal audit: %v", err)
	}

	if restoredContract.ContractID != contract.ContractID || restoredContract.Version != contract.Version {
		t.Fatalf("unexpected restored contract metadata: %+v", restoredContract)
	}
	if !restoredAudit.ReleaseReady {
		t.Fatalf("expected restored audit to stay release ready, got %+v", restoredAudit)
	}
	foundDashboardIDField := false
	for _, field := range restoredContract.DashboardSchema.Fields {
		if field == (ContractField{Name: "dashboard_id", FieldType: "string", Required: true, Description: "Stable dashboard identifier."}) {
			foundDashboardIDField = true
			break
		}
	}
	if !foundDashboardIDField {
		t.Fatalf("expected dashboard_id field after round trip, got %+v", restoredContract.DashboardSchema.Fields)
	}
}
