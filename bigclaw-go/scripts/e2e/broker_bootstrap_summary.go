package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"

	"bigclaw-go/internal/config"
	"bigclaw-go/internal/events"
)

func main() {
	output := flag.String("output", "docs/reports/broker-bootstrap-review-summary.json", "output path for the broker bootstrap review summary")
	flag.Parse()

	cfg := config.LoadFromEnv()
	summary := events.BrokerBootstrapReviewSummaryFromConfig(
		cfg.EventLogBackend,
		cfg.EventLogTargetBackend,
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

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		panic(err)
	}
	body, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(*output, append(body, '\n'), 0o644); err != nil {
		panic(err)
	}
}
