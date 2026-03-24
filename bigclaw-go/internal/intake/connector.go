package intake

import (
	"fmt"
	"strings"
)

type Connector interface {
	Name() string
	FetchIssues(project string, states []string) ([]SourceIssue, error)
}

type GitHubConnector struct{}

func (GitHubConnector) Name() string { return "github" }

func (GitHubConnector) FetchIssues(project string, states []string) ([]SourceIssue, error) {
	return []SourceIssue{
		{
			Source:      "github",
			SourceID:    fmt.Sprintf("%s#1", project),
			Title:       "Fix flaky test",
			Description: "CI flaky on macOS",
			Labels:      []string{"bug", "ci"},
			Priority:    "P1",
			State:       firstState(states),
			Links:       map[string]string{"issue": fmt.Sprintf("https://github.com/%s/issues/1", project)},
		},
	}, nil
}

type LinearConnector struct{}

func (LinearConnector) Name() string { return "linear" }

func (LinearConnector) FetchIssues(project string, states []string) ([]SourceIssue, error) {
	return []SourceIssue{
		{
			Source:      "linear",
			SourceID:    fmt.Sprintf("%s-101", project),
			Title:       "Implement queue persistence",
			Description: "Need restart-safe queue",
			Labels:      []string{"platform"},
			Priority:    "P0",
			State:       firstState(states),
			Links:       map[string]string{"issue": fmt.Sprintf("https://linear.app/%s/issue/%s-101", project, project)},
		},
	}, nil
}

type JiraConnector struct{}

func (JiraConnector) Name() string { return "jira" }

func (JiraConnector) FetchIssues(project string, states []string) ([]SourceIssue, error) {
	return []SourceIssue{
		{
			Source:      "jira",
			SourceID:    fmt.Sprintf("%s-23", project),
			Title:       "Runbook automation",
			Description: "Automate oncall runbook",
			Labels:      []string{"ops"},
			Priority:    "P2",
			State:       firstState(states),
			Links:       map[string]string{"issue": fmt.Sprintf("https://jira.example.com/browse/%s-23", project)},
		},
	}, nil
}

type ClawHostConnector struct{}

func (ClawHostConnector) Name() string { return "clawhost" }

func (ClawHostConnector) FetchIssues(project string, states []string) ([]SourceIssue, error) {
	inventory := []SourceIssue{
		{
			Source:      "clawhost",
			SourceID:    fmt.Sprintf("%s/apps/openagi-control/bots/support-router", project),
			Title:       "support-router bot inventory",
			Description: "ClawHost control-plane inventory for the support-router bot, including lifecycle, owner, and route metadata.",
			Labels:      []string{"inventory", "bot", "control-plane", "production"},
			Priority:    "P1",
			State:       "Running",
			Metadata: map[string]string{
				"provider":           "clawhost",
				"tenant_id":          project,
				"app_id":             "app-openagi-control",
				"app_name":           "openagi-control",
				"bot_id":             "bot-support-router",
				"bot_name":           "support-router",
				"owner":              "platform-ops",
				"domain":             "support.openagi.example",
				"lifecycle_state":    "running",
				"pod_namespace":      "clawhost-openagi",
				"service_name":       "support-router",
				"control_plane_kind": "bot",
			},
			Links: map[string]string{
				"issue":         fmt.Sprintf("https://clawhost.example/%s/apps/openagi-control/bots/support-router", project),
				"control_plane": fmt.Sprintf("https://clawhost.example/%s/apps/openagi-control", project),
			},
		},
		{
			Source:      "clawhost",
			SourceID:    fmt.Sprintf("%s/apps/tenant-admin/bots/release-approver", project),
			Title:       "release-approver bot inventory",
			Description: "ClawHost control-plane inventory for the release-approver bot, including approval-gated lifecycle and tenant route metadata.",
			Labels:      []string{"inventory", "bot", "approval"},
			Priority:    "P2",
			State:       "Pending Approval",
			Metadata: map[string]string{
				"provider":           "clawhost",
				"tenant_id":          project,
				"app_id":             "app-tenant-admin",
				"app_name":           "tenant-admin",
				"bot_id":             "bot-release-approver",
				"bot_name":           "release-approver",
				"owner":              "release-eng",
				"domain":             "release.openagi.example",
				"lifecycle_state":    "pending_approval",
				"pod_namespace":      "clawhost-openagi",
				"service_name":       "release-approver",
				"control_plane_kind": "bot",
			},
			Links: map[string]string{
				"issue":         fmt.Sprintf("https://clawhost.example/%s/apps/tenant-admin/bots/release-approver", project),
				"control_plane": fmt.Sprintf("https://clawhost.example/%s/apps/tenant-admin", project),
			},
		},
	}
	if len(states) == 0 {
		return inventory, nil
	}
	filtered := make([]SourceIssue, 0, len(inventory))
	for _, issue := range inventory {
		if sourceStateMatches(issue.State, states) {
			filtered = append(filtered, issue)
		}
	}
	return filtered, nil
}

func ConnectorByName(name string) (Connector, bool) {
	switch normalizeConnectorName(name) {
	case "github":
		return GitHubConnector{}, true
	case "clawhost":
		return ClawHostConnector{}, true
	case "linear":
		return LinearConnector{}, true
	case "jira":
		return JiraConnector{}, true
	default:
		return nil, false
	}
}

func normalizeConnectorName(name string) string {
	switch name {
	default:
		return lowerTrim(name)
	}
}

func firstState(states []string) string {
	for _, state := range states {
		if trimmed := trim(state); trimmed != "" {
			return trimmed
		}
	}
	return string(SourceStatusTodo)
}

func sourceStateMatches(issueState string, states []string) bool {
	normalizedIssueState := normalizeSourceStatus(issueState)
	for _, candidate := range states {
		normalizedCandidate := normalizeSourceStatus(candidate)
		if normalizedCandidate == "" {
			continue
		}
		if normalizedIssueState == normalizedCandidate || strings.Contains(normalizedIssueState, normalizedCandidate) || strings.Contains(normalizedCandidate, normalizedIssueState) {
			return true
		}
	}
	return false
}
