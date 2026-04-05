package reporting

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestBuildCrossProcessCoordinationSurface(t *testing.T) {
	root := t.TempDir()
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/multi-node-shared-queue-report.json"), map[string]any{
		"count":                     200,
		"cross_node_completions":    99,
		"duplicate_completed_tasks": []any{},
		"duplicate_started_tasks":   []any{},
	})
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/multi-subscriber-takeover-validation-report.json"), map[string]any{
		"summary": map[string]any{
			"scenario_count":           3,
			"passing_scenarios":        3,
			"duplicate_delivery_count": 4,
			"stale_write_rejections":   2,
		},
	})
	writeFixtureJSON(t, filepath.Join(root, "bigclaw-go/docs/reports/live-multi-node-subscriber-takeover-report.json"), map[string]any{
		"summary": map[string]any{
			"scenario_count":         3,
			"passing_scenarios":      3,
			"stale_write_rejections": 3,
		},
	})

	report, err := BuildCrossProcessCoordinationSurface(root, CrossProcessCoordinationSurfaceOptions{
		Now: time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build coordination surface: %v", err)
	}
	if asString(report["status"]) != "local-capability-surface" {
		t.Fatalf("unexpected status: %+v", report)
	}
	readiness := asMap(report["runtime_readiness_levels"])
	if !strings.HasPrefix(asString(readiness["live_proven"]), "Shipped runtime behavior") {
		t.Fatalf("unexpected readiness levels: %+v", readiness)
	}
	summary := asMap(report["summary"])
	if asInt(summary["shared_queue_cross_node_completions"]) != 99 || asInt(summary["takeover_passing_scenarios"]) != 3 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	ceiling := stringSliceFromAny(report["current_ceiling"])
	if len(ceiling) == 0 || ceiling[0] != "no partitioned topic model" {
		t.Fatalf("unexpected current ceiling: %+v", ceiling)
	}
}

func TestCheckedInCoordinationSurfaceMatchesExpectedShape(t *testing.T) {
	root, err := FindRepoRoot("..")
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	report, err := BuildCrossProcessCoordinationSurface(root, CrossProcessCoordinationSurfaceOptions{
		Now: time.Date(2026, 3, 17, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("build coordination surface: %v", err)
	}
	if asInt(asMap(report["summary"])["shared_queue_duplicate_completed_tasks"]) != 0 || asInt(asMap(report["summary"])["takeover_stale_write_rejections"]) != 2 {
		t.Fatalf("unexpected summary: %+v", report["summary"])
	}
	levels := make([]string, 0, len(asMap(report["runtime_readiness_levels"])))
	for key := range asMap(report["runtime_readiness_levels"]) {
		levels = append(levels, key)
	}
	sort.Strings(levels)
	expected := []string{"contract_only", "harness_proven", "live_proven", "supporting_surface"}
	for idx := range expected {
		if levels[idx] != expected[idx] {
			t.Fatalf("unexpected readiness levels: %+v", levels)
		}
	}
	capabilities := anyToMapSlice(report["capabilities"])
	if asString(capabilities[0]["capability"]) != "shared_queue_task_coordination" || asString(capabilities[0]["runtime_readiness"]) != "live_proven" {
		t.Fatalf("unexpected capabilities: %+v", capabilities[0])
	}
}
