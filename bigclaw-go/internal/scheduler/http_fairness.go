package scheduler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type HTTPFairnessStore struct {
	baseURL     *url.URL
	client      *http.Client
	bearerToken string
	mu          sync.RWMutex
	lastError   string
	healthy     bool
}

type fairnessThrottleRequest struct {
	Now      time.Time    `json:"now"`
	TenantID string       `json:"tenant_id"`
	Rules    RoutingRules `json:"rules"`
}

type fairnessThrottleResponse struct {
	ShouldThrottle bool `json:"should_throttle"`
}

type fairnessRecordRequest struct {
	Now      time.Time    `json:"now"`
	TenantID string       `json:"tenant_id"`
	Rules    RoutingRules `json:"rules"`
}

type fairnessSnapshotRequest struct {
	Now   time.Time    `json:"now"`
	Rules RoutingRules `json:"rules"`
}

func NewHTTPFairnessStore(endpoint, bearerToken string) (*HTTPFairnessStore, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid fairness endpoint: %s", endpoint)
	}
	return &HTTPFairnessStore{
		baseURL:     parsed,
		client:      &http.Client{Timeout: 5 * time.Second},
		bearerToken: bearerToken,
		healthy:     true,
	}, nil
}

func (s *HTTPFairnessStore) ShouldThrottle(now time.Time, tenantID string, rules RoutingRules) bool {
	if s == nil || !fairnessEnabled(rules) || strings.TrimSpace(tenantID) == "" {
		return false
	}
	var response fairnessThrottleResponse
	err := s.doJSON(context.Background(), http.MethodPost, "/throttle", fairnessThrottleRequest{Now: now, TenantID: strings.TrimSpace(tenantID), Rules: rules}, &response)
	s.recordStatus(err)
	if err != nil {
		return false
	}
	return response.ShouldThrottle
}

func (s *HTTPFairnessStore) RecordAccepted(now time.Time, tenantID string, rules RoutingRules) {
	if s == nil || !fairnessEnabled(rules) || strings.TrimSpace(tenantID) == "" {
		return
	}
	err := s.doJSON(context.Background(), http.MethodPost, "/record", fairnessRecordRequest{Now: now, TenantID: strings.TrimSpace(tenantID), Rules: rules}, nil)
	s.recordStatus(err)
}

func (s *HTTPFairnessStore) Snapshot(now time.Time, rules RoutingRules) FairnessSnapshot {
	snapshot := fairnessBaseSnapshot(rules, "http", true)
	if s == nil {
		snapshot.Healthy = false
		return snapshot
	}
	snapshot.Endpoint = s.baseURL.String()
	var remote FairnessSnapshot
	err := s.doJSON(context.Background(), http.MethodPost, "/snapshot", fairnessSnapshotRequest{Now: now, Rules: rules}, &remote)
	s.recordStatus(err)
	if err != nil {
		snapshot.Healthy = false
		snapshot.LastError = err.Error()
		return snapshot
	}
	remote.Backend = "http"
	remote.Shared = true
	remote.Healthy = true
	remote.Endpoint = s.baseURL.String()
	return remote
}

func (s *HTTPFairnessStore) recordStatus(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.healthy = false
		s.lastError = err.Error()
		return
	}
	s.healthy = true
	s.lastError = ""
}

func (s *HTTPFairnessStore) doJSON(ctx context.Context, method, path string, body any, out any) error {
	endpoint := s.baseURL.ResolveReference(&url.URL{Path: path})
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
	req.Header.Set("Content-Type", "application/json")
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
		return fmt.Errorf("fairness api status %d: %s", resp.StatusCode, strings.TrimSpace(string(contents)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
