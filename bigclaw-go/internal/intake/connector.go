package intake

import "fmt"

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
	records := sampleClawHostInventory(project)
	issues := make([]SourceIssue, 0, len(records))
	for _, record := range records {
		issues = append(issues, record.SourceIssue(project, states))
	}
	return issues, nil
}

func ConnectorByName(name string) (Connector, bool) {
	switch normalizeConnectorName(name) {
	case "github":
		return GitHubConnector{}, true
	case "linear":
		return LinearConnector{}, true
	case "jira":
		return JiraConnector{}, true
	case "clawhost":
		return ClawHostConnector{}, true
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
