package api

import (
	"bufio"
	"bytes"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type monitorRequest struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	TS     string `json:"ts"`
}

type monitorMinuteBucket struct {
	Minute   int64 `json:"minute"`
	Requests int   `json:"requests"`
	Errors   int   `json:"errors"`
}

type serverMonitor struct {
	startTime      time.Time
	requestTotal   int
	errorTotal     int
	recentRequests []monitorRequest
	minuteBuckets  []monitorMinuteBucket
	mu             sync.Mutex
}

func newServerMonitor(now time.Time) *serverMonitor {
	if now.IsZero() {
		now = time.Now()
	}
	return &serverMonitor{
		startTime:      now,
		recentRequests: make([]monitorRequest, 0, 20),
		minuteBuckets:  make([]monitorMinuteBucket, 0, 5),
	}
}

func (m *serverMonitor) record(path string, status int, now time.Time) {
	if m == nil {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	minute := now.Unix() / 60

	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestTotal++
	if status >= http.StatusBadRequest {
		m.errorTotal++
	}

	m.recentRequests = append(m.recentRequests, monitorRequest{
		Path:   path,
		Status: fmt.Sprintf("%d", status),
		TS:     fmt.Sprintf("%.3f", float64(now.UnixNano())/float64(time.Second)),
	})
	if len(m.recentRequests) > 20 {
		m.recentRequests = append([]monitorRequest(nil), m.recentRequests[len(m.recentRequests)-20:]...)
	}

	lastIndex := len(m.minuteBuckets) - 1
	if lastIndex < 0 || m.minuteBuckets[lastIndex].Minute != minute {
		m.minuteBuckets = append(m.minuteBuckets, monitorMinuteBucket{Minute: minute})
		if len(m.minuteBuckets) > 5 {
			m.minuteBuckets = append([]monitorMinuteBucket(nil), m.minuteBuckets[len(m.minuteBuckets)-5:]...)
		}
		lastIndex = len(m.minuteBuckets) - 1
	}
	m.minuteBuckets[lastIndex].Requests++
	if status >= http.StatusBadRequest {
		m.minuteBuckets[lastIndex].Errors++
	}
}

func (m *serverMonitor) snapshot(now time.Time) monitorSnapshot {
	if m == nil {
		return monitorSnapshot{}
	}
	if now.IsZero() {
		now = time.Now()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	recent := append([]monitorRequest(nil), m.recentRequests...)
	buckets := append([]monitorMinuteBucket(nil), m.minuteBuckets...)
	uptimeSeconds := now.Sub(m.startTime).Seconds()
	if uptimeSeconds < 0 {
		uptimeSeconds = 0
	}
	errorRate := 0.0
	if m.requestTotal > 0 {
		errorRate = float64(m.errorTotal) / float64(m.requestTotal)
	}
	healthSummary := "healthy"
	switch {
	case errorRate >= 0.2:
		healthSummary = "critical"
	case errorRate >= 0.05:
		healthSummary = "degraded"
	}

	return monitorSnapshot{
		Status:               "ok",
		UptimeSeconds:        round3(uptimeSeconds),
		RequestTotal:         m.requestTotal,
		ErrorTotal:           m.errorTotal,
		BigClawUptime:        round3(uptimeSeconds),
		BigClawRequests:      m.requestTotal,
		BigClawErrors:        m.errorTotal,
		BigClawHTTPErrorRate: round4(errorRate),
		HealthSummary:        healthSummary,
		RecentRequests:       recent,
		Rolling5M:            buckets,
	}
}

type monitorSnapshot struct {
	Status               string                `json:"status,omitempty"`
	UptimeSeconds        float64               `json:"uptime_seconds,omitempty"`
	RequestTotal         int                   `json:"request_total,omitempty"`
	ErrorTotal           int                   `json:"error_total,omitempty"`
	BigClawUptime        float64               `json:"bigclaw_uptime_seconds"`
	BigClawRequests      int                   `json:"bigclaw_http_requests_total"`
	BigClawErrors        int                   `json:"bigclaw_http_errors_total"`
	BigClawHTTPErrorRate float64               `json:"bigclaw_http_error_rate"`
	HealthSummary        string                `json:"health_summary"`
	RecentRequests       []monitorRequest      `json:"recent_requests"`
	Rolling5M            []monitorMinuteBucket `json:"rolling_5m"`
}

func (s *Server) monitorSnapshot() monitorSnapshot {
	if s == nil || s.Monitor == nil {
		return monitorSnapshot{}
	}
	return s.Monitor.snapshot(s.Now())
}

func (s *Server) monitorHealthPayload() map[string]any {
	snapshot := s.monitorSnapshot()
	return map[string]any{
		"status":          snapshot.Status,
		"uptime_seconds":  snapshot.UptimeSeconds,
		"request_total":   snapshot.RequestTotal,
		"error_total":     snapshot.ErrorTotal,
		"recent_requests": snapshot.RecentRequests,
		"rolling_5m":      snapshot.Rolling5M,
	}
}

func (s *Server) monitorMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writer := &statusCapturingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		if s != nil && s.Monitor != nil {
			s.Monitor.record(r.URL.Path, writer.status, s.Now())
		}
	})
}

func (m monitorSnapshot) alertsPayload() map[string]any {
	level := "ok"
	message := "System healthy"
	switch {
	case m.BigClawHTTPErrorRate >= 0.2:
		level = "critical"
		message = "High HTTP error rate detected"
	case m.BigClawHTTPErrorRate >= 0.05:
		level = "warn"
		message = "Elevated HTTP error rate detected"
	}
	return map[string]any{
		"level":         level,
		"message":       message,
		"error_rate":    m.BigClawHTTPErrorRate,
		"request_total": m.BigClawRequests,
		"error_total":   m.BigClawErrors,
	}
}

func renderMonitorHTML(snapshot monitorSnapshot) string {
	var recentRows bytes.Buffer
	for _, item := range snapshot.RecentRequests {
		recentRows.WriteString("<tr><td>")
		recentRows.WriteString(html.EscapeString(item.TS))
		recentRows.WriteString("</td><td>")
		recentRows.WriteString(html.EscapeString(item.Path))
		recentRows.WriteString("</td><td>")
		recentRows.WriteString(html.EscapeString(item.Status))
		recentRows.WriteString("</td></tr>")
	}
	if recentRows.Len() == 0 {
		recentRows.WriteString("<tr><td colspan='3'>No requests yet</td></tr>")
	}

	var rollingRows bytes.Buffer
	for _, bucket := range snapshot.Rolling5M {
		rollingRows.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%d</td><td>%d</td></tr>", bucket.Minute, bucket.Requests, bucket.Errors))
	}
	if rollingRows.Len() == 0 {
		rollingRows.WriteString("<tr><td colspan='3'>No rolling data yet</td></tr>")
	}

	return fmt.Sprintf(`<!doctype html>
<html>
<head>
  <meta charset='utf-8'>
  <meta name='viewport' content='width=device-width, initial-scale=1'>
  <title>BigClaw Monitor</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 0; background:#f6f7fb; color:#0f172a; }
    .container { max-width: 1040px; margin: 24px auto; padding: 0 16px; }
    .cards { display:grid; grid-template-columns: repeat(auto-fit,minmax(180px,1fr)); gap:12px; }
    .card { background:#fff; border:1px solid #e2e8f0; border-radius:12px; padding:12px; }
    .label { color:#64748b; font-size:12px; }
    .value { font-size:24px; font-weight:700; margin-top:4px; }
    table { width:100%%; border-collapse: collapse; background:#fff; border:1px solid #e2e8f0; border-radius:12px; overflow:hidden; }
    th,td { border-bottom:1px solid #e2e8f0; padding:8px 10px; text-align:left; font-size:13px; }
    h1,h2 { margin: 0 0 10px; }
    section { margin-top: 16px; }
    .muted { color:#64748b; font-size:12px; }
  </style>
</head>
<body>
  <div class='container'>
    <h1>BigClaw Monitor</h1>
    <p class='muted'>Auto refresh every 5s · endpoint: /metrics.json</p>
    <div class='cards'>
      <div class='card'><div class='label'>Uptime (s)</div><div class='value' id='uptime'>%.3f</div></div>
      <div class='card'><div class='label'>Requests</div><div class='value' id='requests'>%d</div></div>
      <div class='card'><div class='label'>Errors</div><div class='value' id='errors'>%d</div></div>
      <div class='card'><div class='label'>Error Rate</div><div class='value' id='error-rate'>%.4f</div></div>
      <div class='card'><div class='label'>Health</div><div class='value' id='health-summary'>%s</div></div>
    </div>
    <section>
      <h2>Rolling 5m</h2>
      <table id='rolling-table'>
        <thead><tr><th>minute</th><th>requests</th><th>errors</th></tr></thead>
        <tbody>%s</tbody>
      </table>
    </section>
    <section>
      <h2>Recent Requests</h2>
      <table id='recent-table'>
        <thead><tr><th>ts</th><th>path</th><th>status</th></tr></thead>
        <tbody>%s</tbody>
      </table>
    </section>
  </div>
  <script>
    async function refreshMonitor() {
      try {
        const res = await fetch('/metrics.json', { cache: 'no-store' });
        const data = await res.json();
        document.getElementById('uptime').textContent = data.bigclaw_uptime_seconds;
        document.getElementById('requests').textContent = data.bigclaw_http_requests_total;
        document.getElementById('errors').textContent = data.bigclaw_http_errors_total;
        document.getElementById('error-rate').textContent = data.bigclaw_http_error_rate;
        document.getElementById('health-summary').textContent = data.health_summary;

        const rollingBody = document.querySelector('#rolling-table tbody');
        rollingBody.innerHTML = (data.rolling_5m || []).map((b) =>
          "<tr><td>" + b.minute + "</td><td>" + b.requests + "</td><td>" + b.errors + "</td></tr>"
        ).join('') || "<tr><td colspan='3'>No rolling data yet</td></tr>";

        const recentBody = document.querySelector('#recent-table tbody');
        recentBody.innerHTML = (data.recent_requests || []).map((r) =>
          "<tr><td>" + r.ts + "</td><td>" + r.path + "</td><td>" + r.status + "</td></tr>"
        ).join('') || "<tr><td colspan='3'>No requests yet</td></tr>";
      } catch (e) {
        console.error('monitor refresh failed', e);
      }
    }
    setInterval(refreshMonitor, 5000);
  </script>
</body>
</html>`,
		snapshot.BigClawUptime,
		snapshot.BigClawRequests,
		snapshot.BigClawErrors,
		snapshot.BigClawHTTPErrorRate,
		html.EscapeString(snapshot.HealthSummary),
		rollingRows.String(),
		recentRows.String(),
	)
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusCapturingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijacker unsupported")
	}
	return hijacker.Hijack()
}

func (w *statusCapturingResponseWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func (w *statusCapturingResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	if readerFrom, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return readerFrom.ReadFrom(r)
	}
	return io.Copy(w.ResponseWriter, r)
}

func round3(value float64) float64 {
	return float64(int(value*1000+0.5)) / 1000
}

func round4(value float64) float64 {
	return float64(int(value*10000+0.5)) / 10000
}
