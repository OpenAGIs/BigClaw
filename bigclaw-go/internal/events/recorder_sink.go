package events

import (
	"context"

	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/observability"
)

type RecorderSink struct {
	Recorder *observability.Recorder
}

func (s RecorderSink) Write(_ context.Context, event domain.Event) error {
	if s.Recorder != nil {
		s.Recorder.Record(event)
	}
	return nil
}
