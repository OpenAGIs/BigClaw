package intake

import "testing"

func TestConnectorsFetchMinimumIssue(t *testing.T) {
	connectors := []Connector{GitHubConnector{}, LinearConnector{}, JiraConnector{}, ClawHostConnector{}}
	for _, connector := range connectors {
		states := []string{"Todo"}
		if connector.Name() == "clawhost" {
			states = []string{"Running"}
		}
		issues, err := connector.FetchIssues("OpenAGIs/BigClaw", states)
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
	for _, name := range []string{"github", "linear", "jira", "clawhost"} {
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

func TestClawHostConnectorFiltersInventoryByLifecycleState(t *testing.T) {
	connector := ClawHostConnector{}
	issues, err := connector.FetchIssues("openagi", []string{"Running"})
	if err != nil {
		t.Fatalf("fetch clawhost issues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected one running inventory item, got %d", len(issues))
	}
	if issues[0].Source != "clawhost" || issues[0].Metadata["tenant_id"] != "openagi" || issues[0].Metadata["domain"] == "" {
		t.Fatalf("unexpected clawhost issue payload: %+v", issues[0])
	}
}
