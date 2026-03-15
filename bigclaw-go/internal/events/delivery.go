package events

import "bigclaw-go/internal/domain"

func WithDelivery(event domain.Event, mode domain.EventDeliveryMode) domain.Event {
	annotated := event
	annotated.Delivery = &domain.EventDelivery{
		Mode:           mode,
		Replay:         mode == domain.EventDeliveryModeReplay,
		IdempotencyKey: event.ID,
	}
	return annotated
}

func WithDeliveryBatch(items []domain.Event, mode domain.EventDeliveryMode) []domain.Event {
	if len(items) == 0 {
		return nil
	}
	annotated := make([]domain.Event, len(items))
	for index, item := range items {
		annotated[index] = WithDelivery(item, mode)
	}
	return annotated
}
