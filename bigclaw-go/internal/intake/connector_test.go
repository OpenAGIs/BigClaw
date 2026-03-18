package intake

import "testing"

func TestConnectorsFetchIssuesPreserveCanonicalSourceIssueFields(t *testing.T) {
	connectors := []struct {
		connector      Connector
		project        string
		wantSourceID   string
		wantPriority   string
		wantIssueURL   string
		wantLabelCount int
	}{
		{
			connector:      GitHubConnector{},
			project:        "OpenAGIs/BigClaw",
			wantSourceID:   "OpenAGIs/BigClaw#1",
			wantPriority:   "P1",
			wantIssueURL:   "https://github.com/OpenAGIs/BigClaw/issues/1",
			wantLabelCount: 2,
		},
		{
			connector:      LinearConnector{},
			project:        "OpenAGI",
			wantSourceID:   "OpenAGI-101",
			wantPriority:   "P0",
			wantIssueURL:   "https://linear.app/OpenAGI/issue/OpenAGI-101",
			wantLabelCount: 1,
		},
		{
			connector:      JiraConnector{},
			project:        "OPE",
			wantSourceID:   "OPE-23",
			wantPriority:   "P2",
			wantIssueURL:   "https://jira.example.com/browse/OPE-23",
			wantLabelCount: 1,
		},
	}
	for _, item := range connectors {
		connector := item.connector
		issues, err := connector.FetchIssues(item.project, []string{"Todo"})
		if err != nil {
			t.Fatalf("fetch issues for %s: %v", connector.Name(), err)
		}
		if len(issues) == 0 {
			t.Fatalf("expected at least one issue for %s", connector.Name())
		}
		if issues[0].Source != connector.Name() {
			t.Fatalf("expected first issue source %q, got %q", connector.Name(), issues[0].Source)
		}
		if issues[0].SourceID != item.wantSourceID {
			t.Fatalf("expected first issue source_id %q, got %q", item.wantSourceID, issues[0].SourceID)
		}
		if issues[0].Title == "" {
			t.Fatalf("expected title for %s issue", connector.Name())
		}
		if issues[0].Description == "" {
			t.Fatalf("expected description for %s issue", connector.Name())
		}
		if issues[0].Priority != item.wantPriority {
			t.Fatalf("expected priority %q for %s, got %q", item.wantPriority, connector.Name(), issues[0].Priority)
		}
		if issues[0].State != "Todo" {
			t.Fatalf("expected requested state Todo for %s, got %q", connector.Name(), issues[0].State)
		}
		if len(issues[0].Labels) != item.wantLabelCount {
			t.Fatalf("expected %d labels for %s, got %d", item.wantLabelCount, connector.Name(), len(issues[0].Labels))
		}
		if issues[0].Links["issue"] != item.wantIssueURL {
			t.Fatalf("expected issue link %q for %s, got %q", item.wantIssueURL, connector.Name(), issues[0].Links["issue"])
		}
	}
}

func TestConnectorsDefaultStateFallsBackToTodo(t *testing.T) {
	connectors := []Connector{GitHubConnector{}, LinearConnector{}, JiraConnector{}}
	for _, connector := range connectors {
		issues, err := connector.FetchIssues("OpenAGIs/BigClaw", nil)
		if err != nil {
			t.Fatalf("fetch issues for %s: %v", connector.Name(), err)
		}
		if issues[0].State != string(SourceStatusTodo) {
			t.Fatalf("expected default state %q for %s, got %q", SourceStatusTodo, connector.Name(), issues[0].State)
		}
	}
}

func TestConnectorByNameReturnsKnownConnectors(t *testing.T) {
	for _, name := range []string{"github", "linear", "jira"} {
		connector, ok := ConnectorByName(name)
		if !ok {
			t.Fatalf("expected connector %q", name)
		}
		if connector.Name() != name {
			t.Fatalf("expected connector name %q, got %q", name, connector.Name())
		}
	}
	if _, ok := ConnectorByName("unknown"); ok {
		t.Fatalf("expected unknown connector lookup to fail")
	}
}
