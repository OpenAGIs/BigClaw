package repo

import (
	"reflect"
	"testing"
)

func TestRegistryResolvesSpaceChannelAndAgentDeterministically(t *testing.T) {
	registry := NewRegistry()
	registry.RegisterSpace(RepoSpace{
		SpaceID:     "space-1",
		ProjectKey:  "BIGCLAW",
		Repo:        "OpenAGIs/BigClaw",
		SidecarURL:  "http://127.0.0.1:4041",
		HealthState: "healthy",
	})

	resolved, ok := registry.ResolveSpace("BIGCLAW")
	if !ok {
		t.Fatalf("expected repo space to resolve")
	}
	if resolved.Repo != "OpenAGIs/BigClaw" {
		t.Fatalf("unexpected repo space: %+v", resolved)
	}

	if channel := registry.ResolveDefaultChannel("BIGCLAW", "OPE-141"); channel != "bigclaw-ope-141" {
		t.Fatalf("unexpected default channel: %s", channel)
	}

	agent := registry.ResolveAgent("native cloud", "reviewer")
	if agent.RepoAgentID != "agent-native-cloud" {
		t.Fatalf("unexpected repo agent: %+v", agent)
	}
}

func TestRegistryRoundTripAndRepoPlaneNormalization(t *testing.T) {
	registry := NormalizeRegistry(map[string]any{
		"spaces_by_project": map[string]any{
			"BIGCLAW": map[string]any{
				"space_id":                 "s-1",
				"project_key":              "BIGCLAW",
				"repo":                     "OpenAGIs/BigClaw",
				"sidecar_url":              "http://127.0.0.1:4041",
				"health_state":             "healthy",
				"metadata":                 map[string]any{"team": "platform"},
				"sidecar_enabled":          true,
				"default_channel_strategy": "task",
			},
		},
		"agents_by_actor": map[string]any{
			"native cloud": map[string]any{
				"actor":         "native cloud",
				"repo_agent_id": "agent-native-cloud",
				"display_name":  "native cloud",
				"roles":         []any{"executor"},
			},
		},
	})

	space, ok := registry.ResolveSpace("BIGCLAW")
	if !ok {
		t.Fatalf("expected restored space")
	}
	if space.DefaultBranch != "main" || !space.SidecarEnabled || space.HealthState != "healthy" {
		t.Fatalf("unexpected normalized repo space defaults: %+v", space)
	}
	if !reflect.DeepEqual(space.Metadata, map[string]any{"team": "platform"}) {
		t.Fatalf("unexpected repo space metadata: %+v", space.Metadata)
	}

	agent := registry.ResolveAgent("native cloud", "reviewer")
	if agent.RepoAgentID != "agent-native-cloud" {
		t.Fatalf("unexpected restored repo agent: %+v", agent)
	}

	link := NormalizeRunCommitLink(map[string]any{
		"run_id":        "run-143",
		"commit_hash":   "ccc333",
		"role":          "accepted",
		"repo_space_id": "space-1",
		"actor":         "repo-agent",
		"metadata":      map[string]any{"source": "closeout"},
	})
	if link.RunID != "run-143" || link.CommitHash != "ccc333" || link.Role != "accepted" {
		t.Fatalf("unexpected run commit link normalization: %+v", link)
	}
	if !reflect.DeepEqual(link.Metadata, map[string]any{"source": "closeout"}) {
		t.Fatalf("unexpected run commit metadata: %+v", link.Metadata)
	}
}
