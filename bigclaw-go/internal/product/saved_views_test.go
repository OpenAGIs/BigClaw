package product

import (
	"encoding/json"
	"net/url"
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
			parsed, err := url.Parse(view.Route)
			if err != nil {
				t.Fatalf("parse premium view route: %v", err)
			}
			if parsed.Path != "/v2/control-center" || parsed.Query().Get("team") != "platform" || parsed.Query().Get("project") != "apollo" {
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

func TestBuildSavedViewRouteEncodesScopedFilters(t *testing.T) {
	route := buildSavedViewRoute("/v2/control-center", "platform & ops", "apollo/mobile")
	parsed, err := url.Parse(route)
	if err != nil {
		t.Fatalf("parse encoded saved view route: %v", err)
	}
	if parsed.Path != "/v2/control-center" {
		t.Fatalf("unexpected route path: %s", route)
	}
	if parsed.Query().Get("team") != "platform & ops" || parsed.Query().Get("project") != "apollo/mobile" {
		t.Fatalf("expected encoded scope filters to round-trip, got %s", route)
	}
	if strings.Contains(route, "team=platform & ops") || strings.Contains(route, "project=apollo/mobile") {
		t.Fatalf("expected reserved characters to be URL encoded, got %s", route)
	}
}

func TestBuildSavedViewRouteOmitsBlankScopedFilters(t *testing.T) {
	if route := buildSavedViewRoute("/v2/control-center", "   ", ""); route != "/v2/control-center" {
		t.Fatalf("expected base route when scoped filters are blank, got %s", route)
	}

	route := buildSavedViewRoute("/v2/control-center", "  platform  ", "   ")
	parsed, err := url.Parse(route)
	if err != nil {
		t.Fatalf("parse trimmed saved view route: %v", err)
	}
	if parsed.Path != "/v2/control-center" || parsed.Query().Get("team") != "platform" || parsed.Query().Get("project") != "" {
		t.Fatalf("expected trimmed team-only route, got %s", route)
	}
}

func TestViewScopeSuffixSanitizesReservedCharacters(t *testing.T) {
	suffix := viewScopeSuffix("Platform & Ops", "apollo/mobile")
	if suffix != "-platform-ops-apollo-mobile" {
		t.Fatalf("unexpected sanitized scope suffix: %s", suffix)
	}

	catalog := BuildSavedViewCatalog(nil, "alice", "Platform & Ops", "apollo/mobile")
	if len(catalog.Views) == 0 || len(catalog.Subscriptions) == 0 {
		t.Fatalf("expected views and subscriptions in catalog: %+v", catalog)
	}
	for _, view := range catalog.Views {
		if strings.Contains(view.ViewID, " ") || strings.Contains(view.ViewID, "/") || strings.Contains(view.ViewID, "&") {
			t.Fatalf("expected sanitized view id, got %s", view.ViewID)
		}
	}
	for _, subscription := range catalog.Subscriptions {
		if strings.Contains(subscription.SubscriptionID, " ") || strings.Contains(subscription.SubscriptionID, "/") || strings.Contains(subscription.SubscriptionID, "&") {
			t.Fatalf("expected sanitized subscription id, got %s", subscription.SubscriptionID)
		}
	}
}

func TestViewScopeSuffixFallsBackToEmptyWhenScopesCollapse(t *testing.T) {
	if suffix := viewScopeSuffix(" / @ ", "___---"); suffix != "" {
		t.Fatalf("expected empty suffix when scopes collapse to punctuation, got %q", suffix)
	}
}

func TestSavedViewScopeTokenNormalizesMixedSeparators(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: "Platform / Ops @ Night", want: "platform-ops-night"},
		{input: "  Apollo___Mobile---Core  ", want: "apollo-mobile-core"},
		{input: " / @ ", want: ""},
		{input: "", want: ""},
	} {
		if got := savedViewScopeToken(tc.input); got != tc.want {
			t.Fatalf("savedViewScopeToken(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestNormalizedViewOwnerFallsBackToViewer(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{input: "", want: "viewer"},
		{input: "   ", want: "viewer"},
		{input: " alice ", want: "alice"},
	} {
		if got := normalizedViewOwner(tc.input); got != tc.want {
			t.Fatalf("normalizedViewOwner(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestDigestRecipientsDeduplicatesAndLimitsRecipients(t *testing.T) {
	tasks := []domain.Task{
		{Metadata: map[string]string{"owner": " alice ", "reviewer": "bob", "created_by": "carol"}},
		{Metadata: map[string]string{"owner": "dave", "reviewer": "alice", "created_by": "erin"}},
	}

	got := digestRecipients(tasks, " alice ")
	want := []string{"alice", "bob", "carol"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("digestRecipients dedupe/limit = %v, want %v", got, want)
	}
}

func TestDigestRecipientsFallsBackToViewerWhenEmpty(t *testing.T) {
	got := digestRecipients(nil, "   ")
	want := []string{"viewer"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("digestRecipients fallback = %v, want %v", got, want)
	}
}
