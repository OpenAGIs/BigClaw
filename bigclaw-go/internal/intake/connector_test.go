package intake

import "testing"

func TestConnectorsFetchMinimumIssue(t *testing.T) {
	connectors := []Connector{GitHubConnector{}, LinearConnector{}, JiraConnector{}}
	for _, connector := range connectors {
		issues, err := connector.FetchIssues("OpenAGIs/BigClaw", []string{"Todo"})
		if err != nil {
			t.Fatalf("fetch issues for %s: %v", connector.Name(), err)
		}
		if len(issues) == 0 {
			t.Fatalf("expected at least one issue for %s", connector.Name())
		}
		if issues[0].Source != connector.Name() {
			t.Fatalf("expected first issue source %q, got %q", connector.Name(), issues[0].Source)
		}
		if issues[0].Title == "" {
			t.Fatalf("expected title for %s issue", connector.Name())
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
