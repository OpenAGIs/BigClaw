package intake

import "testing"

func TestConnectorsFetchMinimumIssue(t *testing.T) {
	connectors := []Connector{GitHubConnector{}, LinearConnector{}, JiraConnector{}, ClawHostConnector{}}
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
		if connector.Name() == "clawhost" && issues[0].Metadata["control_plane"] != "clawhost" {
			t.Fatalf("expected clawhost metadata in intake issue, got %+v", issues[0])
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

func TestClawHostConnectorFetchesInventoryMetadata(t *testing.T) {
	issues, err := ClawHostConnector{}.FetchIssues("openagi", []string{"running"})
	if err != nil {
		t.Fatalf("fetch clawhost issues: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 clawhost inventory issues, got %d", len(issues))
	}
	first := issues[0]
	if first.Source != "clawhost" || first.State != "running" {
		t.Fatalf("unexpected first clawhost issue: %+v", first)
	}
	if first.Metadata["inventory_kind"] != "claw" || first.Metadata["agent_count"] != "2" || first.Metadata["provider"] == "" || first.Metadata["domain"] == "" {
		t.Fatalf("expected clawhost inventory metadata, got %+v", first.Metadata)
	}
	if first.Links["dashboard"] == "" {
		t.Fatalf("expected clawhost dashboard link, got %+v", first.Links)
	}
}
