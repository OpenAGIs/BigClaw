package product

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
)

func TestBuildSavedViewCatalogAddsScopedViewsAndDigests(t *testing.T) {
	base := time.Date(2026, 3, 19, 9, 0, 0, 0, time.UTC)
	tasks := []domain.Task{
		{ID: "task-1", State: domain.TaskBlocked, RiskLevel: domain.RiskHigh, Metadata: map[string]string{"team": "platform", "project": "apollo", "plan": "premium", "owner": "alice", "reviewer": "bob"}, UpdatedAt: base},
		{ID: "task-2", State: domain.TaskRunning, Metadata: map[string]string{"team": "platform", "project": "apollo", "created_by": "pm-1"}, UpdatedAt: base.Add(time.Minute)},
	}

	catalog := BuildSavedViewCatalog(tasks, "alice", "platform", "apollo")
	if catalog.Name != "operator-console-saved-views" || catalog.Version != "go-v1" {
		t.Fatalf("unexpected catalog metadata: %+v", catalog)
	}
	if len(catalog.Views) < 6 {
		t.Fatalf("expected expanded saved views, got %+v", catalog.Views)
	}
	if catalog.Views[0].Visibility != "team" {
		t.Fatalf("expected scoped visibility, got %+v", catalog.Views[0])
	}

	var premiumFound bool
	for _, view := range catalog.Views {
		if view.ViewID == "premium-runs-platform-apollo" {
			premiumFound = true
			if !strings.Contains(view.Route, "/v2/control-center?team=platform&project=apollo") {
				t.Fatalf("expected premium view route to carry scope, got %+v", view)
			}
		}
	}
	if !premiumFound {
		t.Fatalf("expected premium view in catalog, got %+v", catalog.Views)
	}
	if len(catalog.Subscriptions) != 2 || catalog.Subscriptions[1].SavedViewID != "weekly-ops-platform-apollo" {
		t.Fatalf("unexpected subscriptions: %+v", catalog.Subscriptions)
	}
	if got := strings.Join(catalog.Subscriptions[1].Recipients, ","); got != "alice,bob,pm-1" {
		t.Fatalf("unexpected digest recipients: %s", got)
	}
}

func TestAuditSavedViewCatalogAndRenderReport(t *testing.T) {
	catalog := SavedViewCatalog{
		Name:    "catalog",
		Version: "v1",
		Views: []SavedView{
			{ViewID: "view-1", Name: "Inbox", Route: "/v2/triage/center", Owner: "alice", Visibility: "private", Filters: []SavedViewFilter{{Field: "severity", Operator: "eq", Value: "high"}}, IsDefault: true},
			{ViewID: "view-2", Name: "Inbox", Route: "/v2/triage/center", Owner: "alice", Visibility: "private", Filters: []SavedViewFilter{{Field: "severity", Operator: "eq", Value: "critical"}}, IsDefault: true},
		},
		Subscriptions: []AlertDigestSubscription{
			{SubscriptionID: "sub-1", SavedViewID: "missing", Channel: "sms", Cadence: "monthly"},
		},
	}

	audit := AuditSavedViewCatalog(catalog)
	if audit.ReadinessScore != 0 {
		t.Fatalf("expected readiness score to reflect multiple gaps, got %+v", audit)
	}
	if len(audit.DuplicateViewNames) != 1 || len(audit.DuplicateDefaultViews) != 1 {
		t.Fatalf("expected duplicate detection, got %+v", audit)
	}
	if len(audit.OrphanSubscriptions) != 1 || len(audit.SubscriptionsWithInvalidChannel) != 1 || len(audit.SubscriptionsWithInvalidCadence) != 1 {
		t.Fatalf("expected invalid subscription audit findings, got %+v", audit)
	}

	report := RenderSavedViewReport(catalog, audit)
	if !strings.Contains(report, "# Saved Views & Alert Digests Report") || !strings.Contains(report, "Duplicate view names") || !strings.Contains(report, "sub-1") {
		t.Fatalf("unexpected report output: %s", report)
	}
}

func TestSavedViewCatalogJSONRoundTrip(t *testing.T) {
	catalog := BuildSavedViewCatalog([]domain.Task{{ID: "task-1", State: domain.TaskRunning, Metadata: map[string]string{"owner": "alice"}}}, "alice", "", "")
	payload, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	var restored SavedViewCatalog
	if err := json.Unmarshal(payload, &restored); err != nil {
		t.Fatalf("unmarshal catalog: %v", err)
	}
	if restored.Name != catalog.Name || len(restored.Views) != len(catalog.Views) || restored.Subscriptions[0].SubscriptionID == "" {
		t.Fatalf("unexpected round trip: restored=%+v want=%+v", restored, catalog)
	}
}

func TestSavedViewHelperFunctions(t *testing.T) {
	if got := normalizedViewOwner("  alice "); got != "alice" {
		t.Fatalf("expected trimmed owner alice, got %q", got)
	}
	if got := normalizedViewOwner(" "); got != "viewer" {
		t.Fatalf("expected blank owner to fall back to viewer, got %q", got)
	}

	if got := strings.Join(duplicateStrings([]string{"alpha", " beta ", "alpha", "beta", "", "  "}), ","); got != "alpha,beta" {
		t.Fatalf("expected duplicate strings alpha,beta, got %q", got)
	}

	filters := []SavedViewFilter{
		{Field: "severity", Operator: "=", Value: "high"},
		{Field: "state", Operator: "!=", Value: "done"},
	}
	if got := renderSavedViewFilters(filters); got != "severity=high, state!=done" {
		t.Fatalf("unexpected saved-view filter rendering: %q", got)
	}
	if got := renderSavedViewFilters(nil); got != "none" {
		t.Fatalf("expected empty filters to render as none, got %q", got)
	}

	scopeMap := map[string][]string{
		"team":    {"platform", "apollo"},
		"private": {"alice"},
	}
	if got := renderSavedViewScopeMap(scopeMap); got != "private=alice; team=apollo, platform" {
		t.Fatalf("unexpected saved-view scope map rendering: %q", got)
	}
	if got := renderSavedViewScopeMap(nil); got != "none" {
		t.Fatalf("expected empty scope map to render as none, got %q", got)
	}
}
