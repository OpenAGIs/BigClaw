package events

import (
	"testing"

	"bigclaw-go/internal/domain"
)

func TestWithDeliveryAnnotatesReplayMetadata(t *testing.T) {
	event := domain.Event{ID: "evt-1", Type: domain.EventTaskQueued}

	annotated := WithDelivery(event, domain.EventDeliveryModeReplay)

	if annotated.Delivery == nil {
		t.Fatalf("expected delivery metadata")
	}
	if annotated.Delivery.Mode != domain.EventDeliveryModeReplay {
		t.Fatalf("expected replay mode, got %q", annotated.Delivery.Mode)
	}
	if !annotated.Delivery.Replay {
		t.Fatalf("expected replay marker")
	}
	if annotated.Delivery.IdempotencyKey != "evt-1" {
		t.Fatalf("expected fallback idempotency key, got %q", annotated.Delivery.IdempotencyKey)
	}
}

func TestWithDeliveryBatchPreservesEventOrder(t *testing.T) {
	items := []domain.Event{
		{ID: "evt-1", Type: domain.EventTaskQueued},
		{ID: "evt-2", Type: domain.EventTaskStarted},
	}

	annotated := WithDeliveryBatch(items, domain.EventDeliveryModeLive)

	if len(annotated) != 2 {
		t.Fatalf("expected 2 events, got %d", len(annotated))
	}
	if annotated[0].ID != "evt-1" || annotated[1].ID != "evt-2" {
		t.Fatalf("expected batch order to be preserved")
	}
	if annotated[0].Delivery == nil || annotated[1].Delivery == nil {
		t.Fatalf("expected delivery metadata on every event")
	}
}
