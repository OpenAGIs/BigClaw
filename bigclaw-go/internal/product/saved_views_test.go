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

func TestBuildSavedViewCatalogSortsViewsAndSubscriptions(t *testing.T) {
	tasks := []domain.Task{
		{ID: "task-1", State: domain.TaskRunning, RiskLevel: domain.RiskHigh, Metadata: map[string]string{"plan": "premium", "owner": "alice"}},
	}

	catalog := BuildSavedViewCatalog(tasks, "alice", "platform", "apollo")
	if got := catalog.Views[0].ViewID; got != "active-runs-platform-apollo" {
		t.Fatalf("expected stable sorted view order, first view = %s", got)
	}
	if got := catalog.Views[len(catalog.Views)-1].ViewID; got != "weekly-ops-platform-apollo" {
		t.Fatalf("expected weekly ops to sort last, got %s", got)
	}
	if got := catalog.Subscriptions[0].SubscriptionID; got != "saved-view-daily-triage-platform-apollo" {
		t.Fatalf("expected stable sorted subscription order, got %+v", catalog.Subscriptions)
	}
	if got := catalog.Subscriptions[1].SubscriptionID; got != "saved-view-weekly-ops-platform-apollo" {
		t.Fatalf("expected weekly ops digest second, got %+v", catalog.Subscriptions)
	}
}

func TestBuildSavedViewCatalogPremiumOnlyAddsPremiumView(t *testing.T) {
	tasks := []domain.Task{
		{ID: "task-1", State: domain.TaskRunning, RiskLevel: domain.RiskLow, Metadata: map[string]string{"plan": "premium", "owner": "alice"}},
	}

	catalog := BuildSavedViewCatalog(tasks, "alice", "", "")
	var premiumFound, highRiskFound bool
	for _, view := range catalog.Views {
		if view.ViewID == "premium-runs" {
			premiumFound = true
		}
		if view.ViewID == "high-risk" {
			highRiskFound = true
		}
	}
	if !premiumFound || highRiskFound {
		t.Fatalf("expected only premium optional view branch, got %+v", catalog.Views)
	}
}

func TestBuildSavedViewCatalogHighRiskOnlyAddsHighRiskView(t *testing.T) {
	tasks := []domain.Task{
		{ID: "task-1", State: domain.TaskRunning, RiskLevel: domain.RiskHigh, Metadata: map[string]string{"plan": "standard", "owner": "alice"}},
	}

	catalog := BuildSavedViewCatalog(tasks, "alice", "", "")
	var premiumFound, highRiskFound bool
	for _, view := range catalog.Views {
		if view.ViewID == "premium-runs" {
			premiumFound = true
		}
		if view.ViewID == "high-risk" {
			highRiskFound = true
		}
	}
	if premiumFound || !highRiskFound {
		t.Fatalf("expected only high-risk optional view branch, got %+v", catalog.Views)
	}
}

func TestBuildSavedViewCatalogUnscopedBaseline(t *testing.T) {
	tasks := []domain.Task{
		{ID: "task-1", State: domain.TaskRunning, Metadata: map[string]string{"owner": "alice"}},
	}

	catalog := BuildSavedViewCatalog(tasks, " alice ", "   ", "")
	if len(catalog.Views) != 5 {
		t.Fatalf("expected only baseline saved views without premium/high-risk extras, got %+v", catalog.Views)
	}
	if len(catalog.Subscriptions) != 2 {
		t.Fatalf("expected baseline subscriptions, got %+v", catalog.Subscriptions)
	}
	for _, view := range catalog.Views {
		if strings.Contains(view.ViewID, "-") && (strings.HasSuffix(view.ViewID, "-") || strings.Contains(view.ViewID, "--")) {
			t.Fatalf("unexpected unscoped view id formatting: %s", view.ViewID)
		}
		if view.Visibility != "private" {
			t.Fatalf("expected private visibility for unscoped catalog, got %+v", view)
		}
		if strings.Contains(view.Route, "?") {
			t.Fatalf("expected unscoped route without query string, got %s", view.Route)
		}
	}
	if catalog.Subscriptions[0].SubscriptionID != "saved-view-daily-triage" || catalog.Subscriptions[1].SubscriptionID != "saved-view-weekly-ops" {
		t.Fatalf("unexpected unscoped subscription ids: %+v", catalog.Subscriptions)
	}
}

func TestBuildSavedViewCatalogBaselineFieldContracts(t *testing.T) {
	catalog := BuildSavedViewCatalog(nil, "alice", "", "")
	index := map[string]SavedView{}
	for _, view := range catalog.Views {
		index[view.ViewID] = view
	}

	if !index["active-runs"].Pinned || !index["active-runs"].IsDefault || index["active-runs"].SortBy != "priority:asc,updated_at:desc" {
		t.Fatalf("unexpected active-runs baseline contract: %+v", index["active-runs"])
	}
	if !index["blocked-runs"].Pinned || index["blocked-runs"].SortBy != "updated_at:desc" {
		t.Fatalf("unexpected blocked-runs baseline contract: %+v", index["blocked-runs"])
	}
	if !index["triage-inbox"].Pinned || index["triage-inbox"].SortBy != "severity:desc,updated_at:desc" {
		t.Fatalf("unexpected triage-inbox baseline contract: %+v", index["triage-inbox"])
	}
	if !index["regressions"].Pinned || index["regressions"].SortBy != "severity:desc,updated_at:desc" {
		t.Fatalf("unexpected regressions baseline contract: %+v", index["regressions"])
	}
	if index["weekly-ops"].Pinned || index["weekly-ops"].IsDefault || index["weekly-ops"].SortBy != "week_end:desc" {
		t.Fatalf("unexpected weekly-ops baseline contract: %+v", index["weekly-ops"])
	}
}

func TestBuildSavedViewCatalogProjectScoped(t *testing.T) {
	tasks := []domain.Task{
		{ID: "task-1", State: domain.TaskRunning, Metadata: map[string]string{"owner": "alice"}},
	}

	catalog := BuildSavedViewCatalog(tasks, "alice", "", "apollo/mobile")
	if len(catalog.Views) != 5 {
		t.Fatalf("expected baseline project-scoped views, got %+v", catalog.Views)
	}
	for _, view := range catalog.Views {
		if !strings.Contains(view.ViewID, "-apollo-mobile") {
			t.Fatalf("expected project scope suffix in view id, got %s", view.ViewID)
		}
		if view.Visibility != "team" {
			t.Fatalf("expected project-only scope to promote team visibility, got %+v", view)
		}
		parsed, err := url.Parse(view.Route)
		if err != nil {
			t.Fatalf("parse project-scoped route: %v", err)
		}
		if parsed.Query().Get("team") != "" || parsed.Query().Get("project") != "apollo/mobile" {
			t.Fatalf("expected project-only scoped route, got %s", view.Route)
		}
	}
	if catalog.Subscriptions[0].SubscriptionID != "saved-view-daily-triage-apollo-mobile" || catalog.Subscriptions[1].SubscriptionID != "saved-view-weekly-ops-apollo-mobile" {
		t.Fatalf("unexpected project-scoped subscription ids: %+v", catalog.Subscriptions)
	}
}

func TestBuildSavedViewCatalogTeamScoped(t *testing.T) {
	tasks := []domain.Task{
		{ID: "task-1", State: domain.TaskRunning, Metadata: map[string]string{"owner": "alice"}},
	}

	catalog := BuildSavedViewCatalog(tasks, "alice", "platform", "")
	if len(catalog.Views) != 5 {
		t.Fatalf("expected baseline team-scoped views, got %+v", catalog.Views)
	}
	for _, view := range catalog.Views {
		if !strings.Contains(view.ViewID, "-platform") {
			t.Fatalf("expected team scope suffix in view id, got %s", view.ViewID)
		}
		if view.Visibility != "team" {
			t.Fatalf("expected team-only scope to keep team visibility, got %+v", view)
		}
		parsed, err := url.Parse(view.Route)
		if err != nil {
			t.Fatalf("parse team-scoped route: %v", err)
		}
		if parsed.Query().Get("team") != "platform" || parsed.Query().Get("project") != "" {
			t.Fatalf("expected team-only scoped route, got %s", view.Route)
		}
	}
	if catalog.Subscriptions[0].SubscriptionID != "saved-view-daily-triage-platform" || catalog.Subscriptions[1].SubscriptionID != "saved-view-weekly-ops-platform" {
		t.Fatalf("unexpected team-scoped subscription ids: %+v", catalog.Subscriptions)
	}
}

func TestBuildSavedViewCatalogBlankActorFallsBackToViewer(t *testing.T) {
	catalog := BuildSavedViewCatalog(nil, "   ", "", "")
	if len(catalog.Views) == 0 || len(catalog.Subscriptions) != 2 {
		t.Fatalf("expected baseline catalog with actor fallback, got %+v", catalog)
	}
	for _, view := range catalog.Views {
		if view.Owner != "viewer" {
			t.Fatalf("expected view owner fallback to viewer, got %+v", view)
		}
	}
	if got := strings.Join(catalog.Subscriptions[0].Recipients, ","); got != "viewer" {
		t.Fatalf("expected daily digest recipients to fall back to viewer, got %s", got)
	}
	if got := strings.Join(catalog.Subscriptions[1].Recipients, ","); got != "viewer" {
		t.Fatalf("expected weekly digest recipients to fall back to viewer, got %s", got)
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

func TestRenderSavedViewReportSummarizesViewsAndDigestCoverage(t *testing.T) {
	catalog := SavedViewCatalog{
		Name:    "BigClaw Views",
		Version: "v3",
		Views: []SavedView{
			{
				ViewID:     "view-ops-needs-approval",
				Name:       "Needs Approval",
				Route:      "/operations/overview",
				Owner:      "ops",
				Visibility: "team",
				Filters: []SavedViewFilter{
					{Field: "status", Operator: "=", Value: "needs-approval"},
				},
				SortBy:    "-updated_at",
				Pinned:    true,
				IsDefault: true,
			},
		},
		Subscriptions: []AlertDigestSubscription{
			{
				SubscriptionID: "digest-ops-daily",
				SavedViewID:    "view-ops-needs-approval",
				Channel:        "email",
				Cadence:        "daily",
				Recipients:     []string{"ops@bigclaw.dev"},
			},
		},
	}

	audit := AuditSavedViewCatalog(catalog)
	report := RenderSavedViewReport(catalog, audit)
	for _, want := range []string{
		"# Saved Views & Alert Digests Report",
		"- Saved Views: 1",
		"- Needs Approval: route=/operations/overview owner=ops visibility=team filters=status=needs-approval sort=-updated_at pinned=true default=true",
		"- digest-ops-daily: view=view-ops-needs-approval channel=email cadence=daily recipients=ops@bigclaw.dev include_empty=false muted=false",
		"- Duplicate view names: none",
		"- Orphan subscriptions: none",
	} {
		if !strings.Contains(report, want) {
			t.Fatalf("expected %q in report, got %s", want, report)
		}
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

func TestCountPremiumAndHighRisk(t *testing.T) {
	tasks := []domain.Task{
		{RiskLevel: domain.RiskHigh, Metadata: map[string]string{"plan": "premium"}},
		{RiskLevel: domain.RiskMedium, Metadata: map[string]string{"plan": " PREMIUM "}},
		{RiskLevel: domain.RiskLow, Metadata: map[string]string{"plan": "standard"}},
		{RiskLevel: domain.RiskHigh, Metadata: map[string]string{"plan": ""}},
	}

	if got := countPremium(tasks); got != 2 {
		t.Fatalf("countPremium = %d, want 2", got)
	}
	if got := countHighRisk(tasks); got != 2 {
		t.Fatalf("countHighRisk = %d, want 2", got)
	}
}

func TestVisibilityForScope(t *testing.T) {
	for _, tc := range []struct {
		name    string
		team    string
		project string
		want    string
	}{
		{name: "unscoped", team: "", project: "", want: "private"},
		{name: "trimmed blanks", team: "   ", project: "\n\t", want: "private"},
		{name: "team scope", team: " platform ", project: "", want: "team"},
		{name: "project scope", team: "", project: " apollo ", want: "team"},
	} {
		if got := visibilityForScope(tc.team, tc.project); got != tc.want {
			t.Fatalf("%s: visibilityForScope(%q, %q) = %q, want %q", tc.name, tc.team, tc.project, got, tc.want)
		}
	}
}

func TestDuplicateStringsTrimsIgnoresBlankAndSorts(t *testing.T) {
	got := duplicateStrings([]string{
		" Inbox ",
		"",
		"blocked",
		"Inbox",
		"blocked",
		"   ",
		"triage",
	})
	want := []string{"Inbox", "blocked"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("duplicateStrings = %v, want %v", got, want)
	}
}

func TestRenderSavedViewFilters(t *testing.T) {
	if got := renderSavedViewFilters(nil); got != "none" {
		t.Fatalf("renderSavedViewFilters(nil) = %q, want %q", got, "none")
	}

	got := renderSavedViewFilters([]SavedViewFilter{
		{Field: "state", Operator: "eq", Value: "blocked"},
		{Field: "risk_level", Operator: "eq", Value: "high"},
	})
	want := "stateeqblocked, risk_leveleqhigh"
	if got != want {
		t.Fatalf("renderSavedViewFilters(...) = %q, want %q", got, want)
	}
}

func TestRenderSavedViewScopeMapSortsScopesAndNames(t *testing.T) {
	if got := renderSavedViewScopeMap(nil); got != "none" {
		t.Fatalf("renderSavedViewScopeMap(nil) = %q, want %q", got, "none")
	}

	got := renderSavedViewScopeMap(map[string][]string{
		"/v2/control-center:bob":  {"Weekly Ops", "Active Runs"},
		"/v2/triage/center:alice": {"Regression Follow-up", "Blocked Runs"},
	})
	want := "/v2/control-center:bob=Active Runs, Weekly Ops; /v2/triage/center:alice=Blocked Runs, Regression Follow-up"
	if got != want {
		t.Fatalf("renderSavedViewScopeMap(...) = %q, want %q", got, want)
	}
}

func TestEmptyFallbackTrimsWhitespace(t *testing.T) {
	if got := emptyFallback("  weekly  ", "none"); got != "weekly" {
		t.Fatalf("emptyFallback trims non-empty values = %q, want %q", got, "weekly")
	}
	if got := emptyFallback("   ", "none"); got != "none" {
		t.Fatalf("emptyFallback blank fallback = %q, want %q", got, "none")
	}
}

func TestAuditSavedViewCatalogEmptyCatalogReadiness(t *testing.T) {
	audit := AuditSavedViewCatalog(SavedViewCatalog{Name: "empty", Version: "v1"})
	if audit.ReadinessScore != 0 {
		t.Fatalf("expected empty catalog readiness score to stay at zero, got %+v", audit)
	}
	if audit.DuplicateViewNames != nil || audit.DuplicateDefaultViews != nil {
		t.Fatalf("expected empty duplicate maps to normalize to nil, got %+v", audit)
	}
}

func TestAuditSavedViewCatalogFlagsInvalidVisibilityAndMissingRecipients(t *testing.T) {
	catalog := SavedViewCatalog{
		Name:    "catalog",
		Version: "v1",
		Views: []SavedView{
			{ViewID: "view-1", Name: "Zulu", Route: "/v2/control-center", Owner: "alice", Visibility: "external", Filters: []SavedViewFilter{{Field: "state", Operator: "eq", Value: "running"}}},
			{ViewID: "view-2", Name: "Alpha", Route: "/v2/control-center", Owner: "alice", Visibility: "shared", Filters: []SavedViewFilter{{Field: "state", Operator: "eq", Value: "blocked"}}},
		},
		Subscriptions: []AlertDigestSubscription{
			{SubscriptionID: "sub-2", SavedViewID: "view-2", Channel: "email", Cadence: "daily", Recipients: []string{"alice"}},
			{SubscriptionID: "sub-1", SavedViewID: "view-1", Channel: "slack", Cadence: "daily"},
		},
	}

	audit := AuditSavedViewCatalog(catalog)
	if got := strings.Join(audit.InvalidVisibilityViews, ","); got != "Alpha,Zulu" {
		t.Fatalf("expected sorted invalid visibility findings, got %+v", audit)
	}
	if got := strings.Join(audit.SubscriptionsMissingRecipients, ","); got != "sub-1" {
		t.Fatalf("expected missing-recipient finding, got %+v", audit)
	}
	if audit.ReadinessScore != 0 {
		t.Fatalf("expected invalid visibility and missing recipients to drive readiness to zero, got %+v", audit)
	}
}

func TestAuditSavedViewCatalogReadinessFloorsAtZero(t *testing.T) {
	catalog := SavedViewCatalog{
		Name:    "catalog",
		Version: "v1",
		Views: []SavedView{
			{ViewID: "view-1", Name: "Inbox", Route: "/v2/control-center", Owner: "alice", Visibility: "private"},
		},
		Subscriptions: []AlertDigestSubscription{
			{SubscriptionID: "sub-1", SavedViewID: "missing", Channel: "pager", Cadence: "monthly"},
		},
	}

	audit := AuditSavedViewCatalog(catalog)
	if audit.ReadinessScore != 0 {
		t.Fatalf("expected readiness score to floor at zero, got %+v", audit)
	}
}

func TestAuditSavedViewCatalogRoundsPartialReadiness(t *testing.T) {
	catalog := SavedViewCatalog{
		Name:    "catalog",
		Version: "v1",
		Views: []SavedView{
			{ViewID: "view-1", Name: "Inbox", Route: "/v2/control-center", Owner: "alice", Visibility: "private", Filters: []SavedViewFilter{{Field: "state", Operator: "eq", Value: "running"}}},
			{ViewID: "view-2", Name: "Blocked", Route: "/v2/control-center", Owner: "alice", Visibility: "private"},
			{ViewID: "view-3", Name: "Ops", Route: "/v2/control-center", Owner: "alice", Visibility: "private", Filters: []SavedViewFilter{{Field: "priority", Operator: "eq", Value: "high"}}},
		},
	}

	audit := AuditSavedViewCatalog(catalog)
	if audit.ReadinessScore != 66.7 {
		t.Fatalf("expected rounded partial readiness score, got %+v", audit)
	}
	if got := strings.Join(audit.ViewsMissingFilters, ","); got != "Blocked" {
		t.Fatalf("expected single missing-filter finding, got %+v", audit)
	}
}

func TestAuditSavedViewCatalogValidCatalogScoresPerfectReadiness(t *testing.T) {
	catalog := SavedViewCatalog{
		Name:    "catalog",
		Version: "v1",
		Views: []SavedView{
			{
				ViewID:     "ops-view",
				Name:       "Ops View",
				Route:      "/v2/control-center?project=apollo",
				Owner:      "alice",
				Visibility: "organization",
				Filters:    []SavedViewFilter{{Field: "state", Operator: "eq", Value: "running"}},
				IsDefault:  true,
			},
		},
		Subscriptions: []AlertDigestSubscription{
			{
				SubscriptionID: "ops-digest",
				SavedViewID:    "ops-view",
				Channel:        "webhook",
				Cadence:        "hourly",
				Recipients:     []string{"alice"},
			},
		},
	}

	audit := AuditSavedViewCatalog(catalog)
	if audit.ReadinessScore != 100 {
		t.Fatalf("expected perfect readiness for valid catalog, got %+v", audit)
	}
	if audit.DuplicateViewNames != nil || audit.DuplicateDefaultViews != nil || audit.OrphanSubscriptions != nil || audit.SubscriptionsWithInvalidChannel != nil || audit.SubscriptionsWithInvalidCadence != nil {
		t.Fatalf("expected clean-path audit findings to stay nil, got %+v", audit)
	}
}

func TestAuditSavedViewCatalogCopiesCatalogMetadataAndCounts(t *testing.T) {
	catalog := SavedViewCatalog{
		Name:    "ops-catalog",
		Version: "v2",
		Views: []SavedView{
			{ViewID: "view-1", Name: "Inbox", Route: "/v2/control-center", Owner: "alice", Visibility: "private", Filters: []SavedViewFilter{{Field: "state", Operator: "eq", Value: "running"}}},
			{ViewID: "view-2", Name: "Blocked", Route: "/v2/control-center", Owner: "alice", Visibility: "team", Filters: []SavedViewFilter{{Field: "state", Operator: "eq", Value: "blocked"}}},
		},
		Subscriptions: []AlertDigestSubscription{
			{SubscriptionID: "sub-1", SavedViewID: "view-1", Channel: "email", Cadence: "daily", Recipients: []string{"alice"}},
		},
	}

	audit := AuditSavedViewCatalog(catalog)
	if audit.CatalogName != "ops-catalog" || audit.Version != "v2" || audit.ViewCount != 2 || audit.SubscriptionCount != 1 {
		t.Fatalf("expected audit metadata/counts to match catalog input, got %+v", audit)
	}
}

func TestRenderSavedViewReportEmptyState(t *testing.T) {
	report := RenderSavedViewReport(
		SavedViewCatalog{Name: "empty", Version: "v1"},
		SavedViewCatalogAudit{CatalogName: "empty", Version: "v1"},
	)
	if !strings.Contains(report, "## Saved Views\n\n- None") {
		t.Fatalf("expected empty saved views section, got %s", report)
	}
	if !strings.Contains(report, "## Alert Digests\n\n- None") {
		t.Fatalf("expected empty alert digests section, got %s", report)
	}
	if !strings.Contains(report, "- Duplicate view names: none") || !strings.Contains(report, "- Orphan subscriptions: none") {
		t.Fatalf("expected empty-state gap fallbacks, got %s", report)
	}
}

func TestRound1(t *testing.T) {
	for _, tc := range []struct {
		input float64
		want  float64
	}{
		{input: 0, want: 0},
		{input: 12.34, want: 12.3},
		{input: 12.35, want: 12.4},
		{input: 99.96, want: 100.0},
	} {
		if got := round1(tc.input); got != tc.want {
			t.Fatalf("round1(%v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestMaxFloat(t *testing.T) {
	for _, tc := range []struct {
		left  float64
		right float64
		want  float64
	}{
		{left: 10, right: 2, want: 10},
		{left: -1, right: 0, want: 0},
		{left: 4.5, right: 4.5, want: 4.5},
	} {
		if got := maxFloat(tc.left, tc.right); got != tc.want {
			t.Fatalf("maxFloat(%v, %v) = %v, want %v", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestRenderSavedViewReportPopulatedRowsUseFallbacks(t *testing.T) {
	catalog := SavedViewCatalog{
		Name:    "catalog",
		Version: "v1",
		Views: []SavedView{
			{
				ViewID:     "weekly-ops",
				Name:       "Weekly Ops",
				Route:      "/v2/reports/weekly",
				Owner:      "alice",
				Visibility: "team",
				Filters:    []SavedViewFilter{{Field: "window", Operator: "eq", Value: "7d"}},
			},
		},
		Subscriptions: []AlertDigestSubscription{
			{
				SubscriptionID: "saved-view-weekly-ops",
				SavedViewID:    "weekly-ops",
				Channel:        "email",
				Cadence:        "weekly",
			},
		},
	}
	audit := SavedViewCatalogAudit{
		CatalogName:       "catalog",
		Version:           "v1",
		ViewCount:         1,
		SubscriptionCount: 1,
		ReadinessScore:    100,
	}

	report := RenderSavedViewReport(catalog, audit)
	if !strings.Contains(report, "Weekly Ops: route=/v2/reports/weekly owner=alice visibility=team filters=windoweq7d sort=none") {
		t.Fatalf("expected populated saved view row with sort fallback, got %s", report)
	}
	if !strings.Contains(report, "saved-view-weekly-ops: view=weekly-ops channel=email cadence=weekly recipients=none include_empty=false muted=false") {
		t.Fatalf("expected populated subscription row with recipients fallback, got %s", report)
	}
}

func TestRenderSavedViewReportGapSectionsStaySorted(t *testing.T) {
	report := RenderSavedViewReport(
		SavedViewCatalog{Name: "catalog", Version: "v1"},
		SavedViewCatalogAudit{
			CatalogName:                     "catalog",
			Version:                         "v1",
			DuplicateViewNames:              map[string][]string{"/b:alice": {"Zulu", "Alpha"}, "/a:bob": {"Beta", "Alpha"}},
			DuplicateDefaultViews:           map[string][]string{"/z:viewer": {"Second", "First"}},
			InvalidVisibilityViews:          []string{"Zulu", "Alpha"},
			ViewsMissingFilters:             []string{"Second", "First"},
			OrphanSubscriptions:             []string{"sub-2", "sub-1"},
			SubscriptionsMissingRecipients:  []string{"sub-4", "sub-3"},
			SubscriptionsWithInvalidChannel: []string{"sub-6", "sub-5"},
			SubscriptionsWithInvalidCadence: []string{"sub-8", "sub-7"},
		},
	)
	if !strings.Contains(report, "- Duplicate view names: /a:bob=Alpha, Beta; /b:alice=Alpha, Zulu") {
		t.Fatalf("expected sorted duplicate view names, got %s", report)
	}
	if !strings.Contains(report, "- Duplicate default views: /z:viewer=First, Second") {
		t.Fatalf("expected sorted duplicate default views, got %s", report)
	}
	if !strings.Contains(report, "- Invalid view visibility: Zulu, Alpha") || !strings.Contains(report, "- Subscriptions with invalid cadence: sub-8, sub-7") {
		t.Fatalf("expected gap list rendering to preserve provided ordering, got %s", report)
	}
}
