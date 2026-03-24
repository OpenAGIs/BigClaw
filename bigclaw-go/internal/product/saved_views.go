package product

import (
	"fmt"
	"net/url"
	"sort"
	"strings"

	"bigclaw-go/internal/domain"
)

var (
	validViewVisibility = map[string]struct{}{
		"private":      {},
		"team":         {},
		"organization": {},
	}
	validDigestChannels = map[string]struct{}{
		"email":   {},
		"slack":   {},
		"webhook": {},
	}
	validDigestCadences = map[string]struct{}{
		"hourly": {},
		"daily":  {},
		"weekly": {},
	}
)

type SavedViewFilter struct {
	Field    string `json:"field"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}

type SavedView struct {
	ViewID     string            `json:"view_id"`
	Name       string            `json:"name"`
	Route      string            `json:"route"`
	Owner      string            `json:"owner"`
	Visibility string            `json:"visibility"`
	Filters    []SavedViewFilter `json:"filters,omitempty"`
	SortBy     string            `json:"sort_by,omitempty"`
	Pinned     bool              `json:"pinned"`
	IsDefault  bool              `json:"is_default"`
}

type AlertDigestSubscription struct {
	SubscriptionID      string   `json:"subscription_id"`
	SavedViewID         string   `json:"saved_view_id"`
	Channel             string   `json:"channel"`
	Cadence             string   `json:"cadence"`
	Recipients          []string `json:"recipients,omitempty"`
	IncludeEmptyResults bool     `json:"include_empty_results"`
	Muted               bool     `json:"muted"`
}

type SavedViewCatalog struct {
	Name          string                    `json:"name"`
	Version       string                    `json:"version"`
	Views         []SavedView               `json:"views,omitempty"`
	Subscriptions []AlertDigestSubscription `json:"subscriptions,omitempty"`
}

type SavedViewCatalogAudit struct {
	CatalogName                     string              `json:"catalog_name"`
	Version                         string              `json:"version"`
	ViewCount                       int                 `json:"view_count"`
	SubscriptionCount               int                 `json:"subscription_count"`
	DuplicateViewNames              map[string][]string `json:"duplicate_view_names,omitempty"`
	InvalidVisibilityViews          []string            `json:"invalid_visibility_views,omitempty"`
	ViewsMissingFilters             []string            `json:"views_missing_filters,omitempty"`
	DuplicateDefaultViews           map[string][]string `json:"duplicate_default_views,omitempty"`
	OrphanSubscriptions             []string            `json:"orphan_subscriptions,omitempty"`
	SubscriptionsMissingRecipients  []string            `json:"subscriptions_missing_recipients,omitempty"`
	SubscriptionsWithInvalidChannel []string            `json:"subscriptions_with_invalid_channel,omitempty"`
	SubscriptionsWithInvalidCadence []string            `json:"subscriptions_with_invalid_cadence,omitempty"`
	ReadinessScore                  float64             `json:"readiness_score"`
}

func BuildSavedViewCatalog(tasks []domain.Task, actor, team, project string) SavedViewCatalog {
	actor = normalizedViewOwner(actor)
	scopeSuffix := viewScopeSuffix(team, project)
	views := []SavedView{
		{
			ViewID:     "active-runs" + scopeSuffix,
			Name:       "Active Runs",
			Route:      buildSavedViewRoute("/v2/control-center", team, project),
			Owner:      actor,
			Visibility: visibilityForScope(team, project),
			Filters: []SavedViewFilter{
				{Field: "state", Operator: "in", Value: "queued,leased,running,retrying"},
			},
			SortBy:    "priority:asc,updated_at:desc",
			Pinned:    true,
			IsDefault: true,
		},
		{
			ViewID:     "blocked-runs" + scopeSuffix,
			Name:       "Blocked Runs",
			Route:      buildSavedViewRoute("/v2/control-center", team, project),
			Owner:      actor,
			Visibility: visibilityForScope(team, project),
			Filters: []SavedViewFilter{
				{Field: "state", Operator: "eq", Value: "blocked"},
			},
			SortBy: "updated_at:desc",
			Pinned: true,
		},
		{
			ViewID:     "triage-inbox" + scopeSuffix,
			Name:       "Triage Inbox",
			Route:      buildSavedViewRoute("/v2/triage/center", team, project),
			Owner:      actor,
			Visibility: visibilityForScope(team, project),
			Filters: []SavedViewFilter{
				{Field: "severity", Operator: "in", Value: "critical,high"},
			},
			SortBy: "severity:desc,updated_at:desc",
			Pinned: true,
		},
		{
			ViewID:     "regressions" + scopeSuffix,
			Name:       "Regression Follow-up",
			Route:      buildSavedViewRoute("/v2/regression/center", team, project),
			Owner:      actor,
			Visibility: visibilityForScope(team, project),
			Filters: []SavedViewFilter{
				{Field: "regression_count", Operator: "gt", Value: "0"},
			},
			SortBy: "severity:desc,updated_at:desc",
			Pinned: true,
		},
		{
			ViewID:     "weekly-ops" + scopeSuffix,
			Name:       "Weekly Ops Review",
			Route:      buildSavedViewRoute("/v2/reports/weekly", team, project),
			Owner:      actor,
			Visibility: visibilityForScope(team, project),
			Filters: []SavedViewFilter{
				{Field: "window", Operator: "eq", Value: "7d"},
			},
			SortBy: "week_end:desc",
		},
	}
	if countPremium(tasks) > 0 {
		views = append(views, SavedView{
			ViewID:     "premium-runs" + scopeSuffix,
			Name:       "Premium Runs",
			Route:      buildSavedViewRoute("/v2/control-center", team, project),
			Owner:      actor,
			Visibility: visibilityForScope(team, project),
			Filters: []SavedViewFilter{
				{Field: "plan", Operator: "eq", Value: "premium"},
			},
			SortBy: "updated_at:desc",
		})
	}
	if countHighRisk(tasks) > 0 {
		views = append(views, SavedView{
			ViewID:     "high-risk" + scopeSuffix,
			Name:       "High Risk Review",
			Route:      buildSavedViewRoute("/v2/control-center", team, project),
			Owner:      actor,
			Visibility: visibilityForScope(team, project),
			Filters: []SavedViewFilter{
				{Field: "risk_level", Operator: "eq", Value: "high"},
			},
			SortBy: "priority:asc,updated_at:desc",
		})
	}
	sort.SliceStable(views, func(i, j int) bool { return views[i].ViewID < views[j].ViewID })

	subscriptions := []AlertDigestSubscription{
		{
			SubscriptionID:      "saved-view-daily-triage" + scopeSuffix,
			SavedViewID:         "triage-inbox" + scopeSuffix,
			Channel:             "slack",
			Cadence:             "daily",
			Recipients:          []string{actor},
			IncludeEmptyResults: false,
		},
		{
			SubscriptionID:      "saved-view-weekly-ops" + scopeSuffix,
			SavedViewID:         "weekly-ops" + scopeSuffix,
			Channel:             "email",
			Cadence:             "weekly",
			Recipients:          digestRecipients(tasks, actor),
			IncludeEmptyResults: true,
		},
	}
	sort.SliceStable(subscriptions, func(i, j int) bool { return subscriptions[i].SubscriptionID < subscriptions[j].SubscriptionID })

	return SavedViewCatalog{
		Name:          "operator-console-saved-views",
		Version:       "go-v1",
		Views:         views,
		Subscriptions: subscriptions,
	}
}

func AuditSavedViewCatalog(catalog SavedViewCatalog) SavedViewCatalogAudit {
	audit := SavedViewCatalogAudit{
		CatalogName:           catalog.Name,
		Version:               catalog.Version,
		ViewCount:             len(catalog.Views),
		SubscriptionCount:     len(catalog.Subscriptions),
		DuplicateViewNames:    map[string][]string{},
		DuplicateDefaultViews: map[string][]string{},
	}
	namesByScope := map[string][]string{}
	defaultsByScope := map[string][]string{}
	viewIndex := map[string]SavedView{}
	for _, view := range catalog.Views {
		scope := fmt.Sprintf("%s:%s", view.Route, view.Owner)
		namesByScope[scope] = append(namesByScope[scope], view.Name)
		if view.IsDefault {
			defaultsByScope[scope] = append(defaultsByScope[scope], view.Name)
		}
		if _, ok := validViewVisibility[view.Visibility]; !ok {
			audit.InvalidVisibilityViews = append(audit.InvalidVisibilityViews, view.Name)
		}
		if len(view.Filters) == 0 {
			audit.ViewsMissingFilters = append(audit.ViewsMissingFilters, view.Name)
		}
		viewIndex[view.ViewID] = view
	}
	for scope, names := range namesByScope {
		if duplicates := duplicateStrings(names); len(duplicates) > 0 {
			audit.DuplicateViewNames[scope] = duplicates
		}
	}
	for scope, names := range defaultsByScope {
		if len(names) > 1 {
			audit.DuplicateDefaultViews[scope] = append([]string(nil), names...)
			sort.Strings(audit.DuplicateDefaultViews[scope])
		}
	}
	for _, subscription := range catalog.Subscriptions {
		if _, ok := viewIndex[subscription.SavedViewID]; !ok {
			audit.OrphanSubscriptions = append(audit.OrphanSubscriptions, subscription.SubscriptionID)
		}
		if len(subscription.Recipients) == 0 {
			audit.SubscriptionsMissingRecipients = append(audit.SubscriptionsMissingRecipients, subscription.SubscriptionID)
		}
		if _, ok := validDigestChannels[subscription.Channel]; !ok {
			audit.SubscriptionsWithInvalidChannel = append(audit.SubscriptionsWithInvalidChannel, subscription.SubscriptionID)
		}
		if _, ok := validDigestCadences[subscription.Cadence]; !ok {
			audit.SubscriptionsWithInvalidCadence = append(audit.SubscriptionsWithInvalidCadence, subscription.SubscriptionID)
		}
	}
	sort.Strings(audit.InvalidVisibilityViews)
	sort.Strings(audit.ViewsMissingFilters)
	sort.Strings(audit.OrphanSubscriptions)
	sort.Strings(audit.SubscriptionsMissingRecipients)
	sort.Strings(audit.SubscriptionsWithInvalidChannel)
	sort.Strings(audit.SubscriptionsWithInvalidCadence)
	if len(audit.DuplicateViewNames) == 0 {
		audit.DuplicateViewNames = nil
	}
	if len(audit.DuplicateDefaultViews) == 0 {
		audit.DuplicateDefaultViews = nil
	}
	penalties := len(audit.InvalidVisibilityViews) + len(audit.ViewsMissingFilters) + len(audit.OrphanSubscriptions) + len(audit.SubscriptionsMissingRecipients) + len(audit.SubscriptionsWithInvalidChannel) + len(audit.SubscriptionsWithInvalidCadence)
	if catalog.Views == nil || len(catalog.Views) == 0 {
		audit.ReadinessScore = 0
		return audit
	}
	audit.ReadinessScore = round1(maxFloat(0, 100-(float64(penalties)*100/float64(len(catalog.Views)))))
	return audit
}

func RenderSavedViewReport(catalog SavedViewCatalog, audit SavedViewCatalogAudit) string {
	lines := []string{
		"# Saved Views & Alert Digests Report",
		"",
		fmt.Sprintf("- Name: %s", catalog.Name),
		fmt.Sprintf("- Version: %s", catalog.Version),
		fmt.Sprintf("- Saved Views: %d", audit.ViewCount),
		fmt.Sprintf("- Alert Subscriptions: %d", audit.SubscriptionCount),
		fmt.Sprintf("- Readiness Score: %.1f", audit.ReadinessScore),
		"",
		"## Saved Views",
		"",
	}
	if len(catalog.Views) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, view := range catalog.Views {
			lines = append(lines, fmt.Sprintf("- %s: route=%s owner=%s visibility=%s filters=%s sort=%s pinned=%t default=%t",
				view.Name, view.Route, view.Owner, view.Visibility, renderSavedViewFilters(view.Filters), emptyFallback(view.SortBy, "none"), view.Pinned, view.IsDefault,
			))
		}
	}
	lines = append(lines, "", "## Alert Digests", "")
	if len(catalog.Subscriptions) == 0 {
		lines = append(lines, "- None")
	} else {
		for _, subscription := range catalog.Subscriptions {
			lines = append(lines, fmt.Sprintf("- %s: view=%s channel=%s cadence=%s recipients=%s include_empty=%t muted=%t",
				subscription.SubscriptionID, subscription.SavedViewID, subscription.Channel, subscription.Cadence, emptyFallback(strings.Join(subscription.Recipients, ", "), "none"), subscription.IncludeEmptyResults, subscription.Muted,
			))
		}
	}
	lines = append(lines, "", "## Gaps", "")
	lines = append(lines, fmt.Sprintf("- Duplicate view names: %s", renderSavedViewScopeMap(audit.DuplicateViewNames)))
	lines = append(lines, fmt.Sprintf("- Invalid view visibility: %s", emptyFallback(strings.Join(audit.InvalidVisibilityViews, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Views missing filters: %s", emptyFallback(strings.Join(audit.ViewsMissingFilters, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Duplicate default views: %s", renderSavedViewScopeMap(audit.DuplicateDefaultViews)))
	lines = append(lines, fmt.Sprintf("- Orphan subscriptions: %s", emptyFallback(strings.Join(audit.OrphanSubscriptions, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Subscriptions missing recipients: %s", emptyFallback(strings.Join(audit.SubscriptionsMissingRecipients, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Subscriptions with invalid channel: %s", emptyFallback(strings.Join(audit.SubscriptionsWithInvalidChannel, ", "), "none")))
	lines = append(lines, fmt.Sprintf("- Subscriptions with invalid cadence: %s", emptyFallback(strings.Join(audit.SubscriptionsWithInvalidCadence, ", "), "none")))
	return strings.Join(lines, "\n") + "\n"
}

func buildSavedViewRoute(base, team, project string) string {
	values := url.Values{}
	if team = strings.TrimSpace(team); team != "" {
		values.Set("team", team)
	}
	if project = strings.TrimSpace(project); project != "" {
		values.Set("project", project)
	}
	if len(values) == 0 {
		return base
	}
	return base + "?" + values.Encode()
}

func viewScopeSuffix(team, project string) string {
	parts := make([]string, 0, 2)
	if team = strings.TrimSpace(team); team != "" {
		parts = append(parts, team)
	}
	if project = strings.TrimSpace(project); project != "" {
		parts = append(parts, project)
	}
	if len(parts) == 0 {
		return ""
	}
	return "-" + strings.Join(parts, "-")
}

func visibilityForScope(team, project string) string {
	if strings.TrimSpace(team) != "" || strings.TrimSpace(project) != "" {
		return "team"
	}
	return "private"
}

func normalizedViewOwner(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor == "" {
		return "viewer"
	}
	return actor
}

func digestRecipients(tasks []domain.Task, actor string) []string {
	seen := map[string]struct{}{}
	recipients := make([]string, 0, 4)
	appendRecipient := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		recipients = append(recipients, value)
	}
	appendRecipient(actor)
	for _, task := range tasks {
		appendRecipient(task.Metadata["owner"])
		appendRecipient(task.Metadata["reviewer"])
		appendRecipient(task.Metadata["created_by"])
		if len(recipients) >= 3 {
			break
		}
	}
	if len(recipients) == 0 {
		recipients = append(recipients, "viewer")
	}
	sort.Strings(recipients)
	return recipients
}

func countPremium(tasks []domain.Task) int {
	count := 0
	for _, task := range tasks {
		if strings.EqualFold(strings.TrimSpace(task.Metadata["plan"]), "premium") {
			count++
		}
	}
	return count
}

func countHighRisk(tasks []domain.Task) int {
	count := 0
	for _, task := range tasks {
		if task.RiskLevel == domain.RiskHigh {
			count++
		}
	}
	return count
}

func duplicateStrings(values []string) []string {
	counts := map[string]int{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		counts[value]++
	}
	out := make([]string, 0)
	for value, count := range counts {
		if count > 1 {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func renderSavedViewFilters(filters []SavedViewFilter) string {
	if len(filters) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(filters))
	for _, filter := range filters {
		parts = append(parts, filter.Field+filter.Operator+filter.Value)
	}
	return strings.Join(parts, ", ")
}

func renderSavedViewScopeMap(values map[string][]string) string {
	if len(values) == 0 {
		return "none"
	}
	scopes := make([]string, 0, len(values))
	for scope := range values {
		scopes = append(scopes, scope)
	}
	sort.Strings(scopes)
	parts := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		names := append([]string(nil), values[scope]...)
		sort.Strings(names)
		parts = append(parts, fmt.Sprintf("%s=%s", scope, strings.Join(names, ", ")))
	}
	return strings.Join(parts, "; ")
}

func emptyFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}

func maxFloat(left, right float64) float64 {
	if left > right {
		return left
	}
	return right
}
