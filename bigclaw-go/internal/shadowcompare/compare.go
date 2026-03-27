package shadowcompare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var terminalStates = map[string]struct{}{
	"succeeded":   {},
	"dead_letter": {},
	"cancelled":   {},
	"failed":      {},
}

type CompareOptions struct {
	PrimaryBaseURL     string
	ShadowBaseURL      string
	Task               map[string]any
	Timeout            time.Duration
	HealthTimeout      time.Duration
	Client             *http.Client
	PollInterval       time.Duration
	HealthPollInterval time.Duration
}

func CompareTask(opts CompareOptions) (map[string]any, error) {
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 180 * time.Second
	}
	healthTimeout := opts.HealthTimeout
	if healthTimeout <= 0 {
		healthTimeout = 60 * time.Second
	}
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	healthPollInterval := opts.HealthPollInterval
	if healthPollInterval <= 0 {
		healthPollInterval = time.Second
	}

	if err := waitHealth(client, opts.PrimaryBaseURL, healthTimeout, healthPollInterval); err != nil {
		return nil, err
	}
	if err := waitHealth(client, opts.ShadowBaseURL, healthTimeout, healthPollInterval); err != nil {
		return nil, err
	}

	primaryTask := cloneMap(opts.Task)
	shadowTask := cloneMap(opts.Task)
	baseID := stringValue(opts.Task["id"], fmt.Sprintf("shadow-%d", time.Now().Unix()))
	traceID := stringValue(opts.Task["trace_id"], baseID)
	primaryTask["id"] = baseID + "-primary"
	shadowTask["id"] = baseID + "-shadow"
	primaryTask["trace_id"] = traceID
	shadowTask["trace_id"] = traceID

	if _, err := requestJSON(client, opts.PrimaryBaseURL, http.MethodPost, "/tasks", primaryTask); err != nil {
		return nil, err
	}
	if _, err := requestJSON(client, opts.ShadowBaseURL, http.MethodPost, "/tasks", shadowTask); err != nil {
		return nil, err
	}

	primaryStatus, err := waitTerminal(client, opts.PrimaryBaseURL, stringValue(primaryTask["id"], ""), timeout, pollInterval)
	if err != nil {
		return nil, err
	}
	shadowStatus, err := waitTerminal(client, opts.ShadowBaseURL, stringValue(shadowTask["id"], ""), timeout, pollInterval)
	if err != nil {
		return nil, err
	}

	primaryEventsPayload, err := requestJSON(client, opts.PrimaryBaseURL, http.MethodGet, "/events?task_id="+url.QueryEscape(stringValue(primaryTask["id"], ""))+"&limit=100", nil)
	if err != nil {
		return nil, err
	}
	shadowEventsPayload, err := requestJSON(client, opts.ShadowBaseURL, http.MethodGet, "/events?task_id="+url.QueryEscape(stringValue(shadowTask["id"], ""))+"&limit=100", nil)
	if err != nil {
		return nil, err
	}
	primaryEvents := mapSliceAt(primaryEventsPayload, "events")
	shadowEvents := mapSliceAt(shadowEventsPayload, "events")

	return map[string]any{
		"trace_id": traceID,
		"primary": map[string]any{
			"task_id": stringValue(primaryTask["id"], ""),
			"status":  primaryStatus,
			"events":  primaryEvents,
		},
		"shadow": map[string]any{
			"task_id": stringValue(shadowTask["id"], ""),
			"status":  shadowStatus,
			"events":  shadowEvents,
		},
		"diff": map[string]any{
			"state_equal":              stringValue(primaryStatus["state"], "") == stringValue(shadowStatus["state"], ""),
			"event_count_delta":        len(primaryEvents) - len(shadowEvents),
			"event_types_equal":        stringSlicesEqual(eventTypes(primaryEvents), eventTypes(shadowEvents)),
			"primary_event_types":      eventTypes(primaryEvents),
			"shadow_event_types":       eventTypes(shadowEvents),
			"primary_timeline_seconds": timelineSeconds(primaryEvents),
			"shadow_timeline_seconds":  timelineSeconds(shadowEvents),
		},
	}, nil
}

func WriteReport(path string, report map[string]any) error {
	body, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func waitHealth(client *http.Client, baseURL string, timeout, pollInterval time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	var lastErr error
	for {
		payload, err := requestJSON(client, baseURL, http.MethodGet, "/healthz", nil)
		if err == nil && boolValue(payload["ok"], false) {
			return nil
		}
		if err != nil {
			lastErr = err
		}
		if ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
		case <-time.After(pollInterval):
		}
	}
	return fmt.Errorf("timeout waiting for health on %s: %v", baseURL, lastErr)
}

func waitTerminal(client *http.Client, baseURL, taskID string, timeout, pollInterval time.Duration) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for {
		status, err := requestJSON(client, baseURL, http.MethodGet, "/tasks/"+taskID, nil)
		if err != nil {
			if ctx.Err() != nil {
				return nil, fmt.Errorf("timeout waiting for %s on %s", taskID, baseURL)
			}
		} else if _, ok := terminalStates[stringValue(status["state"], "")]; ok {
			return status, nil
		}
		if ctx.Err() != nil {
			return nil, fmt.Errorf("timeout waiting for %s on %s", taskID, baseURL)
		}
		select {
		case <-ctx.Done():
		case <-time.After(pollInterval):
		}
	}
}

func requestJSON(client *http.Client, baseURL, method, path string, payload map[string]any) (map[string]any, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, strings.TrimRight(baseURL, "/")+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(bodyBytes) == 0 {
		return map[string]any{}, nil
	}
	var decoded map[string]any
	if err := json.Unmarshal(bodyBytes, &decoded); err != nil {
		return nil, err
	}
	return decoded, nil
}

func eventTypes(events []map[string]any) []string {
	types := make([]string, 0, len(events))
	for _, event := range events {
		types = append(types, stringValue(event["type"], ""))
	}
	return types
}

func timelineSeconds(events []map[string]any) float64 {
	if len(events) < 2 {
		return 0
	}
	start := stringValue(events[0]["timestamp"], "")
	end := stringValue(events[len(events)-1]["timestamp"], "")
	if start == "" || end == "" {
		return 0
	}
	startTS, err := time.Parse(time.RFC3339Nano, strings.Replace(start, "Z", "+00:00", 1))
	if err != nil {
		return 0
	}
	endTS, err := time.Parse(time.RFC3339Nano, strings.Replace(end, "Z", "+00:00", 1))
	if err != nil {
		return 0
	}
	seconds := endTS.Sub(startTS).Seconds()
	if seconds < 0 {
		return 0
	}
	return seconds
}

func ExitCode(report map[string]any) int {
	diff := nestedMap(report, "diff")
	if boolValue(diff["state_equal"], false) && boolValue(diff["event_types_equal"], false) {
		return 0
	}
	return 1
}

func cloneMap(source map[string]any) map[string]any {
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func mapSliceAt(source map[string]any, key string) []map[string]any {
	raw, _ := source[key].([]any)
	items := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if mapped, ok := item.(map[string]any); ok {
			items = append(items, mapped)
		}
	}
	return items
}

func nestedMap(source map[string]any, key string) map[string]any {
	value, _ := source[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func stringValue(value any, fallback string) string {
	text, ok := value.(string)
	if !ok {
		return fallback
	}
	return text
}

func boolValue(value any, fallback bool) bool {
	typed, ok := value.(bool)
	if ok {
		return typed
	}
	return fallback
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func LoadTask(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var task map[string]any
	if err := json.Unmarshal(body, &task); err != nil {
		return nil, err
	}
	if len(task) == 0 {
		return nil, errors.New("task file was empty")
	}
	return task, nil
}
