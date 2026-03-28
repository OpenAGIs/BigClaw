package service

import (
	"encoding/json"
	"fmt"
	"html"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RepoGovernancePolicy struct {
	MaxBundleBytes int
	MaxPushPerHour int
	MaxDiffPerHour int
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

func (e *RepoGovernanceEnforcer) Evaluate(action string, bundleBytes int, sidecarAvailable bool) RepoGovernanceResult {
	if e.policy.SidecarRequired && !sidecarAvailable {
		return RepoGovernanceResult{Allowed: false, Mode: "degraded", Reason: "repo sidecar unavailable"}
	}
	switch action {
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

type requestSample struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	TS     string `json:"ts"`
}

type minuteBucket struct {
	Minute   int64 `json:"minute"`
	Requests int   `json:"requests"`
	Errors   int   `json:"errors"`
}

type ServerMonitoring struct {
	startTime      time.Time
	requestTotal   int
	errorTotal     int
	recentRequests []requestSample
	minuteBuckets  []minuteBucket
	mu             sync.Mutex
}

func NewServerMonitoring() *ServerMonitoring {
	return &ServerMonitoring{startTime: time.Now()}
}

func (m *ServerMonitoring) Record(path string, status int) {
	now := time.Now()
	minute := now.Unix() / 60
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestTotal++
	if status >= http.StatusBadRequest {
		m.errorTotal++
	}
	m.recentRequests = append(m.recentRequests, requestSample{
		Path:   path,
		Status: fmt.Sprintf("%d", status),
		TS:     fmt.Sprintf("%.3f", float64(now.UnixNano())/1e9),
	})
	if len(m.recentRequests) > 20 {
		m.recentRequests = append([]requestSample(nil), m.recentRequests[len(m.recentRequests)-20:]...)
	}
	if len(m.minuteBuckets) == 0 || m.minuteBuckets[len(m.minuteBuckets)-1].Minute != minute {
		m.minuteBuckets = append(m.minuteBuckets, minuteBucket{Minute: minute})
		if len(m.minuteBuckets) > 5 {
			m.minuteBuckets = append([]minuteBucket(nil), m.minuteBuckets[len(m.minuteBuckets)-5:]...)
		}
	}
	last := len(m.minuteBuckets) - 1
	m.minuteBuckets[last].Requests++
	if status >= http.StatusBadRequest {
		m.minuteBuckets[last].Errors++
	}
}

func (m *ServerMonitoring) RequestTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestTotal
}

func (m *ServerMonitoring) HealthPayload() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]any{
		"status":         "ok",
		"uptime_seconds": roundSeconds(time.Since(m.startTime).Seconds()),
		"request_total":  m.requestTotal,
		"error_total":    m.errorTotal,
		"recent_requests": append([]requestSample(nil), m.recentRequests...),
		"rolling_5m":      append([]minuteBucket(nil), m.minuteBuckets...),
	}
}

func (m *ServerMonitoring) MetricsPayload() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	errorRate := 0.0
	if m.requestTotal > 0 {
		errorRate = float64(m.errorTotal) / float64(m.requestTotal)
	}
	summary := "healthy"
	if errorRate >= 0.2 {
		summary = "critical"
	} else if errorRate >= 0.05 {
		summary = "degraded"
	}
	return map[string]any{
		"bigclaw_uptime_seconds":      roundSeconds(time.Since(m.startTime).Seconds()),
		"bigclaw_http_requests_total": m.requestTotal,
		"bigclaw_http_errors_total":   m.errorTotal,
		"bigclaw_http_error_rate":     roundRate(errorRate),
		"health_summary":              summary,
		"recent_requests":             append([]requestSample(nil), m.recentRequests...),
		"rolling_5m":                  append([]minuteBucket(nil), m.minuteBuckets...),
	}
}

func (m *ServerMonitoring) AlertsPayload() map[string]any {
	metrics := m.MetricsPayload()
	errorRate := metrics["bigclaw_http_error_rate"].(float64)
	level := "ok"
	message := "System healthy"
	if errorRate >= 0.2 {
		level = "critical"
		message = "High HTTP error rate detected"
	} else if errorRate >= 0.05 {
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

func (m *ServerMonitoring) MetricsText() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	body := fmt.Sprintf(
		"# HELP bigclaw_uptime_seconds process uptime in seconds\n# TYPE bigclaw_uptime_seconds gauge\nbigclaw_uptime_seconds %.3f\n# HELP bigclaw_http_requests_total total HTTP requests\n# TYPE bigclaw_http_requests_total counter\nbigclaw_http_requests_total %d\n# HELP bigclaw_http_errors_total total HTTP error responses (>=400)\n# TYPE bigclaw_http_errors_total counter\nbigclaw_http_errors_total %d\n",
		time.Since(m.startTime).Seconds(),
		m.requestTotal,
		m.errorTotal,
	)
	for _, bucket := range m.minuteBuckets {
		body += fmt.Sprintf("bigclaw_http_requests_minute{minute=\"%d\"} %d\n", bucket.Minute, bucket.Requests)
		body += fmt.Sprintf("bigclaw_http_errors_minute{minute=\"%d\"} %d\n", bucket.Minute, bucket.Errors)
	}
	return body
}

func NewHandler(directory string, monitoring *ServerMonitoring) (http.Handler, error) {
	if monitoring == nil {
		monitoring = NewServerMonitoring()
	}
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	fileHandler := http.FileServer(http.Dir(absDir))
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, monitoring.HealthPayload())
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		body := monitoring.MetricsText()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/metrics.json", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, monitoring.MetricsPayload())
	})
	mux.HandleFunc("/alerts", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, monitoring.AlertsPayload())
	})
	mux.HandleFunc("/monitor", func(w http.ResponseWriter, _ *http.Request) {
		body := renderMonitorPage(monitoring.MetricsPayload())
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	})
	mux.Handle("/", fileHandler)
	return &monitoringHandler{next: mux, monitoring: monitoring}, nil
}

func CreateServer(host string, port int, directory string) (*http.Server, net.Listener, *ServerMonitoring, error) {
	monitoring := NewServerMonitoring()
	handler, err := NewHandler(directory, monitoring)
	if err != nil {
		return nil, nil, nil, err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return nil, nil, nil, err
	}
	return &http.Server{Handler: handler}, listener, monitoring, nil
}

type monitoringHandler struct {
	next       http.Handler
	monitoring *ServerMonitoring
}

func (h *monitoringHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
	h.next.ServeHTTP(recorder, r)
	h.monitoring.Record(r.URL.Path, recorder.status)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func renderMonitorPage(stats map[string]any) string {
	return fmt.Sprintf(`<!doctype html>
<html>
<head><meta charset='utf-8'><meta name='viewport' content='width=device-width, initial-scale=1'><title>BigClaw Monitor</title></head>
<body>
<h1>BigClaw Monitor</h1>
<p>Auto refresh every 5s</p>
<div>Requests: %v</div>
<div>Error Rate: %v</div>
<div>Health: %v</div>
</body>
</html>`,
		html.EscapeString(fmt.Sprint(stats["bigclaw_http_requests_total"])),
		html.EscapeString(fmt.Sprint(stats["bigclaw_http_error_rate"])),
		html.EscapeString(fmt.Sprint(stats["health_summary"])),
	)
}

func roundSeconds(value float64) float64 {
	return float64(int(value*1000+0.5)) / 1000
}

func roundRate(value float64) float64 {
	return float64(int(value*10000+0.5)) / 10000
}

func EnsureDirectory(path string) error {
	return os.MkdirAll(path, 0o755)
}
