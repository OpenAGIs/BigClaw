package events

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type WebhookConfig struct {
	URLs        []string
	BearerToken string
	Timeout     time.Duration
}

type WebhookSink struct {
	client *http.Client
	cfg    WebhookConfig
}

func NewWebhookSink(cfg WebhookConfig) *WebhookSink {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &WebhookSink{client: &http.Client{Timeout: cfg.Timeout}, cfg: cfg}
}

func (s *WebhookSink) Write(ctx context.Context, event domain.Event) error {
	if len(s.cfg.URLs) == 0 {
		return nil
	}
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	var lastErr error
	for _, endpoint := range s.cfg.URLs {
		if strings.TrimSpace(endpoint) == "" {
			continue
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		if s.cfg.BearerToken != "" {
			req.Header.Set("Authorization", "Bearer "+s.cfg.BearerToken)
		}
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("webhook status %d", resp.StatusCode)
			continue
		}
	}
	return lastErr
}
