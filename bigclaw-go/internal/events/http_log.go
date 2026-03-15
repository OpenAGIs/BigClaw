package events

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
)

type HTTPEventLog struct {
	baseURL     *url.URL
	client      *http.Client
	bearerToken string
}

type checkpointAckRequest struct {
	EventID string    `json:"event_id"`
	AckedAt time.Time `json:"acked_at"`
}

type checkpointResponse struct {
	Checkpoint SubscriberCheckpoint `json:"checkpoint"`
}

type remoteEventsResponse struct {
	Events []domain.Event `json:"events"`
}

type retentionWatermarkResponse struct {
	RetentionWatermark RetentionWatermark `json:"retention_watermark"`
}

func NewHTTPEventLog(endpoint, bearerToken string) (*HTTPEventLog, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid event log endpoint: %s", endpoint)
	}
	return &HTTPEventLog{
		baseURL:     parsed,
		client:      &http.Client{Timeout: 5 * time.Second},
		bearerToken: bearerToken,
	}, nil
}

func (s *HTTPEventLog) Backend() string {
	return "http"
}

func (s *HTTPEventLog) Capabilities() BackendCapabilities {
	return BackendCapabilities{
		Backend:    "http",
		Scope:      "remote_service",
		Publish:    FeatureSupport{Supported: true, Mode: "append_only"},
		Replay:     FeatureSupport{Supported: true, Mode: "durable"},
		Checkpoint: FeatureSupport{Supported: true, Mode: "subscriber_ack"},
		Dedup:      FeatureSupport{Supported: true, Mode: "remote_service", Detail: "Consumer dedup records are expected to persist in the shared event service."},
		Filtering:  FeatureSupport{Supported: true, Mode: "server_side"},
		Retention:  FeatureSupport{Supported: true, Mode: "remote_service", Detail: "Retention boundaries are served by the remote event-log service and may include persisted truncation state."},
	}
}

func (s *HTTPEventLog) Write(ctx context.Context, event domain.Event) error {
	return s.doJSON(ctx, http.MethodPost, "/record", event, nil)
}

func (s *HTTPEventLog) Replay(limit int) ([]domain.Event, error) {
	return s.queryEvents(url.Values{"limit": []string{strconv.Itoa(limit)}})
}

func (s *HTTPEventLog) ReplayAfter(afterID string, limit int) ([]domain.Event, error) {
	params := url.Values{"after_id": []string{afterID}, "limit": []string{strconv.Itoa(limit)}}
	return s.queryEvents(params)
}

func (s *HTTPEventLog) EventsByTask(taskID string, limit int) ([]domain.Event, error) {
	params := url.Values{"task_id": []string{taskID}, "limit": []string{strconv.Itoa(limit)}}
	return s.queryEvents(params)
}

func (s *HTTPEventLog) EventsByTaskAfter(taskID string, afterID string, limit int) ([]domain.Event, error) {
	params := url.Values{"task_id": []string{taskID}, "after_id": []string{afterID}, "limit": []string{strconv.Itoa(limit)}}
	return s.queryEvents(params)
}

func (s *HTTPEventLog) EventsByTrace(traceID string, limit int) ([]domain.Event, error) {
	params := url.Values{"trace_id": []string{traceID}, "limit": []string{strconv.Itoa(limit)}}
	return s.queryEvents(params)
}

func (s *HTTPEventLog) EventsByTraceAfter(traceID string, afterID string, limit int) ([]domain.Event, error) {
	params := url.Values{"trace_id": []string{traceID}, "after_id": []string{afterID}, "limit": []string{strconv.Itoa(limit)}}
	return s.queryEvents(params)
}

func (s *HTTPEventLog) Acknowledge(subscriberID string, eventID string, at time.Time) (SubscriberCheckpoint, error) {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	var response checkpointResponse
	err := s.doJSON(context.Background(), http.MethodPost, "/checkpoints/"+url.PathEscape(strings.TrimSpace(subscriberID)), checkpointAckRequest{EventID: strings.TrimSpace(eventID), AckedAt: at}, &response)
	if err != nil {
		return SubscriberCheckpoint{}, mapRemoteEventLogError(err)
	}
	return response.Checkpoint, nil
}

func (s *HTTPEventLog) Checkpoint(subscriberID string) (SubscriberCheckpoint, error) {
	var response checkpointResponse
	err := s.doJSON(context.Background(), http.MethodGet, "/checkpoints/"+url.PathEscape(strings.TrimSpace(subscriberID)), nil, &response)
	if err != nil {
		return SubscriberCheckpoint{}, mapRemoteEventLogError(err)
	}
	return response.Checkpoint, nil
}

func (s *HTTPEventLog) RetentionWatermark() (RetentionWatermark, error) {
	var response retentionWatermarkResponse
	err := s.doJSON(context.Background(), http.MethodGet, "/watermark", nil, &response)
	if err != nil {
		return RetentionWatermark{}, mapRemoteEventLogError(err)
	}
	return response.RetentionWatermark, nil
}

func (s *HTTPEventLog) Path() string {
	if s == nil || s.baseURL == nil {
		return ""
	}
	return s.baseURL.String()
}

func (s *HTTPEventLog) Close() error {
	return nil
}

func (s *HTTPEventLog) queryEvents(params url.Values) ([]domain.Event, error) {
	if params == nil {
		params = url.Values{}
	}
	if limit := params.Get("limit"); limit == "0" {
		params.Del("limit")
	}
	path := "/events"
	if encoded := params.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var response remoteEventsResponse
	err := s.doJSON(context.Background(), http.MethodGet, path, nil, &response)
	if err != nil {
		return nil, err
	}
	return response.Events, nil
}

func (s *HTTPEventLog) doJSON(ctx context.Context, method, path string, body any, out any) error {
	endpoint, err := s.endpoint(path)
	if err != nil {
		return err
	}
	var payload io.Reader
	if body != nil {
		contents, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(contents)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), payload)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.bearerToken)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		contents, _ := io.ReadAll(resp.Body)
		return statusError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(contents))}
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *HTTPEventLog) endpoint(path string) (*url.URL, error) {
	if s == nil || s.baseURL == nil {
		return nil, fmt.Errorf("event log endpoint unavailable")
	}
	base := *s.baseURL
	requestPath := path
	if requestPath == "" {
		requestPath = "/"
	}
	if index := strings.Index(requestPath, "?"); index >= 0 {
		base.RawQuery = requestPath[index+1:]
		requestPath = requestPath[:index]
	} else {
		base.RawQuery = ""
	}
	base.Path = strings.TrimRight(base.Path, "/") + requestPath
	return &base, nil
}

func mapRemoteEventLogError(err error) error {
	var status statusError
	if !errorsAsStatus(err, &status) {
		return err
	}
	if status.StatusCode == http.StatusNotFound {
		return sql.ErrNoRows
	}
	return err
}

func errorsAsStatus(err error, out *statusError) bool {
	status, ok := err.(statusError)
	if !ok {
		return false
	}
	*out = status
	return true
}

type statusError struct {
	StatusCode int
	Message    string
}

func (e statusError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("event log api status %d", e.StatusCode)
	}
	return fmt.Sprintf("event log api status %d: %s", e.StatusCode, e.Message)
}

var _ EventLog = (*HTTPEventLog)(nil)
var _ CheckpointStore = (*HTTPEventLog)(nil)
