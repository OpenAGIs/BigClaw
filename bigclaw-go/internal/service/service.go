package service

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RepoGovernancePolicy struct {
	MaxBundleBytes  int64
	MaxPushPerHour  int
	MaxDiffPerHour  int
	SidecarRequired bool
}

type RepoGovernanceResult struct {
	Allowed bool   `json:"allowed"`
	Mode    string `json:"mode"`
	Reason  string `json:"reason,omitempty"`
}

type RepoGovernanceEnforcer struct {
	policy    RepoGovernancePolicy
	pushCount int
	diffCount int
}

func NewRepoGovernanceEnforcer(policy RepoGovernancePolicy) *RepoGovernanceEnforcer {
	return &RepoGovernanceEnforcer{policy: policy}
}

func (e *RepoGovernanceEnforcer) Evaluate(action string, bundleBytes int64, sidecarAvailable bool) RepoGovernanceResult {
	if e.policy.SidecarRequired && !sidecarAvailable {
		return RepoGovernanceResult{Allowed: false, Mode: "degraded", Reason: "repo sidecar unavailable"}
	}

	switch strings.TrimSpace(action) {
	case "push":
		if bundleBytes > e.policy.MaxBundleBytes {
			return RepoGovernanceResult{Allowed: false, Mode: "blocked", Reason: "bundle exceeds max size"}
		}
		if e.pushCount >= e.policy.MaxPushPerHour {
			return RepoGovernanceResult{Allowed: false, Mode: "blocked", Reason: "push quota exceeded"}
		}
		e.pushCount++
	case "diff":
		if e.diffCount >= e.policy.MaxDiffPerHour {
			return RepoGovernanceResult{Allowed: false, Mode: "blocked", Reason: "diff quota exceeded"}
		}
		e.diffCount++
	}

	return RepoGovernanceResult{Allowed: true, Mode: "allow"}
}

type requestRecord struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	TS     string `json:"ts"`
}

type minuteBucket struct {
	Minute   int64 `json:"minute"`
	Requests int   `json:"requests"`
	Errors   int   `json:"errors"`
}

type Monitoring struct {
	startTime      time.Time
	requestTotal   int
	errorTotal     int
	recentRequests []requestRecord
	minuteBuckets  []minuteBucket
	now            func() time.Time
	mu             sync.Mutex
}

func NewMonitoring() *Monitoring {
	return &Monitoring{
		startTime:      time.Now(),
		recentRequests: make([]requestRecord, 0, 20),
		minuteBuckets:  make([]minuteBucket, 0, 5),
		now:            time.Now,
	}
}

func (m *Monitoring) RequestTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestTotal
}

func (m *Monitoring) Record(path string, status int) {
	now := m.now()
	minute := now.Unix() / 60

	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestTotal++
	if status >= http.StatusBadRequest {
		m.errorTotal++
	}

	m.recentRequests = append(m.recentRequests, requestRecord{
		Path:   path,
		Status: strconv.Itoa(status),
		TS:     fmt.Sprintf("%.3f", float64(now.UnixNano())/1e9),
	})
	if len(m.recentRequests) > 20 {
		m.recentRequests = append([]requestRecord(nil), m.recentRequests[len(m.recentRequests)-20:]...)
	}

	if len(m.minuteBuckets) == 0 || m.minuteBuckets[len(m.minuteBuckets)-1].Minute != minute {
		m.minuteBuckets = append(m.minuteBuckets, minuteBucket{Minute: minute})
		if len(m.minuteBuckets) > 5 {
			m.minuteBuckets = append([]minuteBucket(nil), m.minuteBuckets[len(m.minuteBuckets)-5:]...)
		}
	}
	last := &m.minuteBuckets[len(m.minuteBuckets)-1]
	last.Requests++
	if status >= http.StatusBadRequest {
		last.Errors++
	}
}

func (m *Monitoring) HealthPayload() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{
		"status":          "ok",
		"uptime_seconds":  roundSeconds(m.now().Sub(m.startTime).Seconds()),
		"request_total":   m.requestTotal,
		"error_total":     m.errorTotal,
		"recent_requests": cloneRecentRequests(m.recentRequests),
		"rolling_5m":      cloneMinuteBuckets(m.minuteBuckets),
	}
}

func (m *Monitoring) MetricsPayload() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	errorRate := 0.0
	if m.requestTotal != 0 {
		errorRate = float64(m.errorTotal) / float64(m.requestTotal)
	}

	summary := "healthy"
	switch {
	case errorRate >= 0.2:
		summary = "critical"
	case errorRate >= 0.05:
		summary = "degraded"
	}

	return map[string]any{
		"bigclaw_uptime_seconds":      roundSeconds(m.now().Sub(m.startTime).Seconds()),
		"bigclaw_http_requests_total": m.requestTotal,
		"bigclaw_http_errors_total":   m.errorTotal,
		"bigclaw_http_error_rate":     roundRate(errorRate),
		"health_summary":              summary,
		"recent_requests":             cloneRecentRequests(m.recentRequests),
		"rolling_5m":                  cloneMinuteBuckets(m.minuteBuckets),
	}
}

func (m *Monitoring) AlertsPayload() map[string]any {
	metrics := m.MetricsPayload()
	errorRate, _ := metrics["bigclaw_http_error_rate"].(float64)
	level := "ok"
	message := "System healthy"
	switch {
	case errorRate >= 0.2:
		level = "critical"
		message = "High HTTP error rate detected"
	case errorRate >= 0.05:
		level = "warn"
		message = "Elevated HTTP error rate detected"
	}
	return map[string]any{
		"level":         level,
		"message":       message,
		"error_rate":    errorRate,
		"request_total": metrics["bigclaw_http_requests_total"],
		"error_total":   metrics["bigclaw_http_errors_total"],
	}
}

func (m *Monitoring) MetricsText() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	lines := []string{
		"# HELP bigclaw_uptime_seconds process uptime in seconds",
		"# TYPE bigclaw_uptime_seconds gauge",
		fmt.Sprintf("bigclaw_uptime_seconds %.3f", m.now().Sub(m.startTime).Seconds()),
		"# HELP bigclaw_http_requests_total total HTTP requests",
		"# TYPE bigclaw_http_requests_total counter",
		fmt.Sprintf("bigclaw_http_requests_total %d", m.requestTotal),
		"# HELP bigclaw_http_errors_total total HTTP error responses (>=400)",
		"# TYPE bigclaw_http_errors_total counter",
		fmt.Sprintf("bigclaw_http_errors_total %d", m.errorTotal),
	}
	for _, bucket := range m.minuteBuckets {
		lines = append(lines,
			fmt.Sprintf("bigclaw_http_requests_minute{minute=%q} %d", strconv.FormatInt(bucket.Minute, 10), bucket.Requests),
			fmt.Sprintf("bigclaw_http_errors_minute{minute=%q} %d", strconv.FormatInt(bucket.Minute, 10), bucket.Errors),
		)
	}
	return strings.Join(lines, "\n") + "\n"
}

func Handler(directory string, monitoring *Monitoring) (http.Handler, error) {
	if monitoring == nil {
		monitoring = NewMonitoring()
	}
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(absDir); err != nil {
		return nil, err
	}

	fileServer := http.FileServer(http.Dir(absDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		switch r.URL.Path {
		case "/health":
			writeJSON(recorder, monitoring.HealthPayload())
		case "/metrics":
			body := monitoring.MetricsText()
			recorder.Header().Set("Content-Type", "text/plain; version=0.0.4")
			recorder.WriteHeader(http.StatusOK)
			_, _ = recorder.Write([]byte(body))
		case "/metrics.json":
			writeJSON(recorder, monitoring.MetricsPayload())
		case "/alerts":
			writeJSON(recorder, monitoring.AlertsPayload())
		case "/monitor":
			recorder.Header().Set("Content-Type", "text/html; charset=utf-8")
			recorder.WriteHeader(http.StatusOK)
			_, _ = recorder.Write([]byte(monitorPage(monitoring.MetricsPayload())))
		default:
			fileServer.ServeHTTP(recorder, r)
		}

		monitoring.Record(r.URL.Path, recorder.status)
	}), nil
}

func monitorPage(stats map[string]any) string {
	recentRows := "<tr><td colspan='3'>No requests yet</td></tr>"
	if recent, ok := stats["recent_requests"].([]requestRecord); ok && len(recent) > 0 {
		rows := make([]string, 0, len(recent))
		for _, item := range recent {
			rows = append(rows, fmt.Sprintf("<tr><td>%s</td><td>%s</td><td>%s</td></tr>", html.EscapeString(item.TS), html.EscapeString(item.Path), html.EscapeString(item.Status)))
		}
		recentRows = strings.Join(rows, "")
	}

	rollingRows := "<tr><td colspan='3'>No rolling data yet</td></tr>"
	if buckets, ok := stats["rolling_5m"].([]minuteBucket); ok && len(buckets) > 0 {
		rows := make([]string, 0, len(buckets))
		for _, bucket := range buckets {
			rows = append(rows, fmt.Sprintf("<tr><td>%d</td><td>%d</td><td>%d</td></tr>", bucket.Minute, bucket.Requests, bucket.Errors))
		}
		rollingRows = strings.Join(rows, "")
	}

	return fmt.Sprintf(`<!doctype html>
<html>
<head>
  <meta charset='utf-8'>
  <meta name='viewport' content='width=device-width, initial-scale=1'>
  <title>BigClaw Monitor</title>
</head>
<body>
  <h1>BigClaw Monitor</h1>
  <p>Auto refresh every 5s</p>
  <div>
    <div>Requests</div>
    <div>Error Rate</div>
    <div>Health</div>
  </div>
  <section>
    <h2>Rolling 5m</h2>
    <table><tbody>%s</tbody></table>
  </section>
  <section>
    <h2>Recent Requests</h2>
    <table><tbody>%s</tbody></table>
  </section>
</body>
</html>`, rollingRows, recentRows)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, payload map[string]any) {
	body, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func roundSeconds(value float64) float64 {
	return float64(int(value*1000+0.5)) / 1000
}

func roundRate(value float64) float64 {
	return float64(int(value*10000+0.5)) / 10000
}

func cloneRecentRequests(in []requestRecord) []requestRecord {
	return append([]requestRecord(nil), in...)
}

func cloneMinuteBuckets(in []minuteBucket) []minuteBucket {
	return append([]minuteBucket(nil), in...)
}
