package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type RepoGovernancePolicy struct {
	MaxBundleBytes  int64 `json:"max_bundle_bytes"`
	MaxPushPerHour  int   `json:"max_push_per_hour"`
	MaxDiffPerHour  int   `json:"max_diff_per_hour"`
	SidecarRequired bool  `json:"sidecar_required"`
}

type GovernanceDecision struct {
	Allowed bool   `json:"allowed"`
	Mode    string `json:"mode,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type RepoGovernanceEnforcer struct {
	policy      RepoGovernancePolicy
	pushCount   int
	diffCount   int
	decisionMux sync.Mutex
}

func NewRepoGovernanceEnforcer(policy RepoGovernancePolicy) *RepoGovernanceEnforcer {
	return &RepoGovernanceEnforcer{policy: policy}
}

func (e *RepoGovernanceEnforcer) Evaluate(action string, bundleBytes int64, sidecarAvailable bool) GovernanceDecision {
	e.decisionMux.Lock()
	defer e.decisionMux.Unlock()

	if e.policy.SidecarRequired && !sidecarAvailable {
		return GovernanceDecision{Allowed: false, Mode: "degraded", Reason: "sidecar unavailable"}
	}
	if e.policy.MaxBundleBytes > 0 && bundleBytes > e.policy.MaxBundleBytes {
		return GovernanceDecision{Allowed: false, Mode: "blocked", Reason: "bundle exceeds configured size quota"}
	}

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "push":
		if e.policy.MaxPushPerHour > 0 && e.pushCount >= e.policy.MaxPushPerHour {
			return GovernanceDecision{Allowed: false, Mode: "blocked", Reason: "push quota exceeded"}
		}
		e.pushCount++
	case "diff":
		if e.policy.MaxDiffPerHour > 0 && e.diffCount >= e.policy.MaxDiffPerHour {
			return GovernanceDecision{Allowed: false, Mode: "blocked", Reason: "diff quota exceeded"}
		}
		e.diffCount++
	}
	return GovernanceDecision{Allowed: true, Mode: "allowed", Reason: "within governance limits"}
}

type RequestRecord struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	TS     string `json:"ts"`
}

type Monitoring struct {
	startTime      time.Time
	requestTotal   int
	errorTotal     int
	recentRequests []RequestRecord
	minuteBuckets  []map[string]int
	mu             sync.Mutex
}

func NewMonitoring() *Monitoring {
	return &Monitoring{startTime: time.Now()}
}

func (m *Monitoring) record(path string, status int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	minute := int(now.Unix() / 60)
	m.requestTotal++
	if status >= 400 {
		m.errorTotal++
	}
	if len(m.recentRequests) >= 20 {
		m.recentRequests = m.recentRequests[1:]
	}
	m.recentRequests = append(m.recentRequests, RequestRecord{
		Path:   path,
		Status: fmt.Sprintf("%d", status),
		TS:     fmt.Sprintf("%.3f", float64(now.UnixNano())/1e9),
	})
	if len(m.minuteBuckets) == 0 || m.minuteBuckets[len(m.minuteBuckets)-1]["minute"] != minute {
		if len(m.minuteBuckets) >= 5 {
			m.minuteBuckets = m.minuteBuckets[1:]
		}
		m.minuteBuckets = append(m.minuteBuckets, map[string]int{"minute": minute, "requests": 0, "errors": 0})
	}
	bucket := m.minuteBuckets[len(m.minuteBuckets)-1]
	bucket["requests"]++
	if status >= 400 {
		bucket["errors"]++
	}
}

func (m *Monitoring) snapshot() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	uptime := time.Since(m.startTime).Seconds()
	errorRate := 0.0
	if m.requestTotal > 0 {
		errorRate = float64(m.errorTotal) / float64(m.requestTotal)
	}
	rolling := make([]map[string]int, 0, len(m.minuteBuckets))
	for _, bucket := range m.minuteBuckets {
		rolling = append(rolling, map[string]int{
			"minute":   bucket["minute"],
			"requests": bucket["requests"],
			"errors":   bucket["errors"],
		})
	}
	recent := append([]RequestRecord(nil), m.recentRequests...)
	healthSummary := "healthy"
	if errorRate >= 0.2 {
		healthSummary = "critical"
	} else if errorRate >= 0.05 {
		healthSummary = "degraded"
	}
	return map[string]any{
		"status":                      "ok",
		"uptime_seconds":              round3(uptime),
		"request_total":               m.requestTotal,
		"error_total":                 m.errorTotal,
		"bigclaw_uptime_seconds":      round3(uptime),
		"bigclaw_http_requests_total": m.requestTotal,
		"bigclaw_http_errors_total":   m.errorTotal,
		"bigclaw_http_error_rate":     round4(errorRate),
		"health_summary":              healthSummary,
		"recent_requests":             recent,
		"rolling_5m":                  rolling,
	}
}

func (m *Monitoring) RequestTotal() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.requestTotal
}

func NewHandler(directory string, monitoring *Monitoring) http.Handler {
	if monitoring == nil {
		monitoring = NewMonitoring()
	}
	fileServer := http.FileServer(http.Dir(directory))
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, status int, payload any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		monitoring.record(r.URL.Path, http.StatusOK)
		snapshot := monitoring.snapshot()
		writeJSON(w, http.StatusOK, map[string]any{
			"status":          snapshot["status"],
			"uptime_seconds":  snapshot["uptime_seconds"],
			"request_total":   snapshot["request_total"],
			"error_total":     snapshot["error_total"],
			"recent_requests": snapshot["recent_requests"],
			"rolling_5m":      snapshot["rolling_5m"],
		})
	})
	mux.HandleFunc("/metrics.json", func(w http.ResponseWriter, r *http.Request) {
		monitoring.record(r.URL.Path, http.StatusOK)
		writeJSON(w, http.StatusOK, monitoring.snapshot())
	})
	mux.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		monitoring.record(r.URL.Path, http.StatusOK)
		snapshot := monitoring.snapshot()
		level := "ok"
		message := "System healthy"
		errorRate := snapshot["bigclaw_http_error_rate"].(float64)
		if errorRate >= 0.2 {
			level = "critical"
			message = "High HTTP error rate detected"
		} else if errorRate >= 0.05 {
			level = "warn"
			message = "Elevated HTTP error rate detected"
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"level":         level,
			"message":       message,
			"error_rate":    errorRate,
			"request_total": snapshot["bigclaw_http_requests_total"],
			"error_total":   snapshot["bigclaw_http_errors_total"],
		})
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		monitoring.record(r.URL.Path, http.StatusOK)
		snapshot := monitoring.snapshot()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = fmt.Fprintf(w, "# HELP bigclaw_uptime_seconds process uptime in seconds\n")
		_, _ = fmt.Fprintf(w, "# TYPE bigclaw_uptime_seconds gauge\n")
		_, _ = fmt.Fprintf(w, "bigclaw_uptime_seconds %.3f\n", snapshot["bigclaw_uptime_seconds"])
		_, _ = fmt.Fprintf(w, "# HELP bigclaw_http_requests_total total HTTP requests\n")
		_, _ = fmt.Fprintf(w, "# TYPE bigclaw_http_requests_total counter\n")
		_, _ = fmt.Fprintf(w, "bigclaw_http_requests_total %v\n", snapshot["bigclaw_http_requests_total"])
		_, _ = fmt.Fprintf(w, "# HELP bigclaw_http_errors_total total HTTP error responses (>=400)\n")
		_, _ = fmt.Fprintf(w, "# TYPE bigclaw_http_errors_total counter\n")
		_, _ = fmt.Fprintf(w, "bigclaw_http_errors_total %v\n", snapshot["bigclaw_http_errors_total"])
	})
	mux.HandleFunc("/monitor", func(w http.ResponseWriter, r *http.Request) {
		monitoring.record(r.URL.Path, http.StatusOK)
		snapshot := monitoring.snapshot()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<!doctype html><html><head><title>BigClaw Monitor</title></head><body><h1>BigClaw Monitor</h1><p>Auto refresh every 5s</p><div>Requests: %v</div><div>Error Rate: %v</div></body></html>", snapshot["bigclaw_http_requests_total"], snapshot["bigclaw_http_error_rate"])
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		fileServer.ServeHTTP(recorder, r)
		monitoring.record(r.URL.Path, recorder.status)
	})
	return mux
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func EnsureStaticIndex(directory string) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	indexPath := filepath.Join(directory, "index.html")
	if _, err := os.Stat(indexPath); err == nil {
		return nil
	}
	return os.WriteFile(indexPath, []byte("<h1>ok</h1>"), 0o644)
}

func round3(value float64) float64 {
	return float64(int(value*1000+0.5)) / 1000
}

func round4(value float64) float64 {
	return float64(int(value*10000+0.5)) / 10000
}
