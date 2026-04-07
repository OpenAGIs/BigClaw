package domain

import (
	"testing"
	"time"
)

func TestNewConsumerDedupKeyProducesStableStorageKeyAndFingerprint(t *testing.T) {
	event := Event{
		ID:        "evt-42",
		Type:      EventTaskCompleted,
		TaskID:    "task-1",
		TraceID:   "trace-1",
		RunID:     "run-1",
		Timestamp: time.Unix(1700000000, 0).UTC(),
	}

	key := NewConsumerDedupKey(" consumer-a ", event)
	if got, want := key.StorageKey(), "v1/consumer-a/evt-42"; got != want {
		t.Fatalf("expected storage key %q, got %q", want, got)
	}

	if got := key.Fingerprint(); got == "" {
		t.Fatalf("expected non-empty fingerprint")
	}

	normalized := ConsumerDedupKey{
		ConsumerID: "consumer-a",
		EventID:    "evt-42",
		EventType:  EventTaskCompleted,
		TaskID:     "task-1",
		TraceID:    "trace-1",
		RunID:      "run-1",
	}
	if got, want := key.Fingerprint(), normalized.Fingerprint(); got != want {
		t.Fatalf("expected normalized fingerprint %q, got %q", want, got)
	}
}

func TestConsumerDedupKeyValidateRequiresConsumerAndEventID(t *testing.T) {
	if err := (ConsumerDedupKey{}).Validate(); err == nil || err.Error() != "consumer dedup key requires consumer_id" {
		t.Fatalf("expected consumer_id validation error, got %v", err)
	}

	if err := (ConsumerDedupKey{ConsumerID: "consumer-a"}).Validate(); err == nil || err.Error() != "consumer dedup key requires event_id" {
		t.Fatalf("expected event_id validation error, got %v", err)
	}

	key := ConsumerDedupKey{
		Version:    " ",
		ConsumerID: " consumer-a ",
		EventID:    " evt-42 ",
		EventType:  EventTaskCompleted,
		TaskID:     " task-1 ",
		TraceID:    " trace-1 ",
		RunID:      " run-1 ",
	}
	if err := key.Validate(); err != nil {
		t.Fatalf("expected normalized key to validate, got %v", err)
	}
	if got, want := key.Normalize().StorageKey(), "v1/consumer-a/evt-42"; got != want {
		t.Fatalf("expected normalized storage key %q, got %q", want, got)
	}
}

func TestConsumerDedupResultStableFingerprintIgnoresMetadataOrder(t *testing.T) {
	first := ConsumerDedupResult{
		Handler:           "audit-sink",
		AppliedAt:         time.Unix(1700000100, 0).UTC(),
		EffectID:          "effect-1",
		EffectSequence:    2,
		EffectFingerprint: "fp-1",
		Summary:           "applied",
		Metadata: map[string]string{
			"tenant": "alpha",
			"mode":   "replay",
		},
	}
	second := ConsumerDedupResult{
		Handler:           "audit-sink",
		AppliedAt:         time.Unix(1700000100, 0).UTC(),
		EffectID:          "effect-1",
		EffectSequence:    2,
		EffectFingerprint: "fp-1",
		Summary:           "applied",
		Metadata: map[string]string{
			"mode":   "replay",
			"tenant": "alpha",
		},
	}

	if got, want := first.StableFingerprint(), second.StableFingerprint(); got != want {
		t.Fatalf("expected equal stable fingerprints, got %q and %q", got, want)
	}
}
