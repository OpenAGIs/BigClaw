package regression

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"bigclaw-go/internal/config"
	"bigclaw-go/internal/events"
)

func TestDurabilityRolloutScorecardReportStaysAligned(t *testing.T) {
	repoRoot := repoRoot(t)
	reportPath := filepath.Join(repoRoot, "docs", "reports", "durability-rollout-scorecard.json")

	var report events.RolloutScorecard
	readJSONFile(t, reportPath, &report)

	cfg := config.Default()
	plan := events.NewDurabilityPlanWithBrokerConfig(
		cfg.EventLogBackend,
		cfg.EventLogTargetBackend,
		cfg.EventLogReplicationFactor,
		events.BrokerRuntimeConfig{
			Driver:             cfg.EventLogBrokerDriver,
			URLs:               cfg.EventLogBrokerURLs,
			Topic:              cfg.EventLogBrokerTopic,
			ConsumerGroup:      cfg.EventLogConsumerGroup,
			PublishTimeout:     cfg.EventLogPublishTimeout,
			ReplayLimit:        cfg.EventLogReplayLimit,
			CheckpointInterval: cfg.EventLogCheckpointInterval,
		},
	)
	want := plan.RolloutScorecard()
	if !reflect.DeepEqual(report, want) {
		t.Fatalf("durability rollout scorecard drifted\nreport=%+v\nwant=%+v", report, want)
	}

	cases := []struct {
		path       string
		substrings []string
	}{
		{
			path: "docs/reports/event-bus-reliability-report.md",
			substrings: []string{
				"event_durability_rollout",
				"durability-rollout-scorecard.json",
			},
		},
		{
			path: "docs/reports/replicated-event-log-durability-rollout-contract.md",
			substrings: []string{
				"event_durability_rollout",
				"durability-rollout-scorecard.json",
			},
		},
	}

	for _, tc := range cases {
		contents := readRepoFile(t, repoRoot, tc.path)
		for _, needle := range tc.substrings {
			if !strings.Contains(contents, needle) {
				t.Fatalf("%s missing substring %q", tc.path, needle)
			}
		}
	}
}
