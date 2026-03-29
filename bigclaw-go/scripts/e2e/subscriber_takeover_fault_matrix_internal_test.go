package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSubscriberTakeoverFaultMatrixBuildReportPreservesSummaryAndScenarioSchema(t *testing.T) {
	repoRoot := subscriberTakeoverRepoRoot(t)
	report, err := buildSubscriberTakeoverReport(repoRoot, defaultSubscriberTakeoverTemplatePath, time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildSubscriberTakeoverReport: %v", err)
	}

	if got := report["generated_at"]; got != "2026-03-29T12:00:00Z" {
		t.Fatalf("generated_at = %v, want 2026-03-29T12:00:00Z", got)
	}
	if got := report["ticket"]; got != "OPE-269" {
		t.Fatalf("ticket = %v, want OPE-269", got)
	}
	if got := report["status"]; got != "local-executable" {
		t.Fatalf("status = %v, want local-executable", got)
	}

	currentPrimitives := asSubscriberTakeoverMap(report["current_primitives"])
	takeoverHarness := asSubscriberTakeoverSlice(currentPrimitives["takeover_harness"])
	if len(takeoverHarness) != 2 || takeoverHarness[0] != goSubscriberTakeoverScriptPath {
		t.Fatalf("unexpected takeover_harness: %+v", takeoverHarness)
	}

	summary := asSubscriberTakeoverMap(report["summary"])
	if asSubscriberTakeoverInt(summary["scenario_count"]) != 3 ||
		asSubscriberTakeoverInt(summary["passing_scenarios"]) != 3 ||
		asSubscriberTakeoverInt(summary["failing_scenarios"]) != 0 ||
		asSubscriberTakeoverInt(summary["duplicate_delivery_count"]) != 4 ||
		asSubscriberTakeoverInt(summary["stale_write_rejections"]) != 2 {
		t.Fatalf("unexpected summary: %+v", summary)
	}

	scenarios := asSubscriberTakeoverSlice(report["scenarios"])
	wantScenarioIDs := []string{
		"takeover-after-primary-crash",
		"lease-expiry-stale-writer-rejected",
		"split-brain-dual-replay-window",
	}
	if len(scenarios) != len(wantScenarioIDs) {
		t.Fatalf("len(scenarios) = %d, want %d", len(scenarios), len(wantScenarioIDs))
	}
	for index, want := range wantScenarioIDs {
		scenario := asSubscriberTakeoverMap(scenarios[index])
		if scenario["id"] != want {
			t.Fatalf("scenario[%d].id = %v, want %s", index, scenario["id"], want)
		}
		for _, key := range []string{
			"lease_owner_timeline",
			"checkpoint_before",
			"checkpoint_after",
			"replay_start_cursor",
			"replay_end_cursor",
			"duplicate_events",
			"audit_log_paths",
			"event_log_excerpt",
			"assertion_results",
			"all_assertions_passed",
		} {
			if _, ok := scenario[key]; !ok {
				t.Fatalf("scenario[%d] missing key %q", index, key)
			}
		}
	}
}

func TestSubscriberTakeoverFaultMatrixPrettyWriterStaysJSONCompatible(t *testing.T) {
	repoRoot := subscriberTakeoverRepoRoot(t)
	report, err := buildSubscriberTakeoverReport(repoRoot, defaultSubscriberTakeoverTemplatePath, time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("buildSubscriberTakeoverReport: %v", err)
	}

	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded["generated_at"] != "2026-03-29T12:00:00Z" {
		t.Fatalf("decoded generated_at = %v", decoded["generated_at"])
	}
}

func subscriberTakeoverRepoRoot(t *testing.T) string {
	t.Helper()
	repoRoot, err := repoRootFromSubscriberTakeoverScript(subscriberTakeoverScriptFilePath())
	if err != nil {
		t.Fatalf("repoRootFromSubscriberTakeoverScript: %v", err)
	}
	return repoRoot
}

func asSubscriberTakeoverMap(value any) map[string]any {
	if cast, ok := value.(map[string]any); ok {
		return cast
	}
	return map[string]any{}
}

func asSubscriberTakeoverSlice(value any) []any {
	if cast, ok := value.([]any); ok {
		return cast
	}
	return nil
}

func asSubscriberTakeoverInt(value any) int {
	switch cast := value.(type) {
	case int:
		return cast
	case int64:
		return int(cast)
	case float64:
		return int(cast)
	default:
		return 0
	}
}
