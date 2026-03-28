package service

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStaticServerMonitoringEndpoints(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "index.html")
	if err := os.WriteFile(indexPath, []byte("<h1>ok</h1>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}

	monitoring := NewMonitoring(timeNow())
	server := httptest.NewServer(NewStaticServerHandler(tempDir, monitoring))
	defer server.Close()

	body := mustGetText(t, server.URL+"/")
	if !strings.Contains(body, "ok") {
		t.Fatalf("expected root body to contain ok, got %q", body)
	}

	var health map[string]any
	mustGetJSON(t, server.URL+"/health", &health)
	if health["status"] != "ok" {
		t.Fatalf("expected status ok, got %+v", health)
	}
	if got, ok := health["request_total"].(float64); !ok || got < 1 {
		t.Fatalf("expected request_total >= 1, got %+v", health["request_total"])
	}

	metrics := mustGetText(t, server.URL+"/metrics")
	if !strings.Contains(metrics, "bigclaw_http_requests_total") || !strings.Contains(metrics, "bigclaw_uptime_seconds") {
		t.Fatalf("unexpected metrics body: %s", metrics)
	}

	var metricsJSON map[string]any
	mustGetJSON(t, server.URL+"/metrics.json", &metricsJSON)
	for _, key := range []string{"bigclaw_http_requests_total", "recent_requests", "rolling_5m", "bigclaw_http_error_rate", "health_summary"} {
		if _, ok := metricsJSON[key]; !ok {
			t.Fatalf("expected metrics json key %q in %+v", key, metricsJSON)
		}
	}

	var alerts map[string]any
	mustGetJSON(t, server.URL+"/alerts", &alerts)
	level, _ := alerts["level"].(string)
	if level != "ok" && level != "warn" && level != "critical" {
		t.Fatalf("unexpected alert level: %+v", alerts)
	}
	if _, ok := alerts["error_rate"]; !ok {
		t.Fatalf("expected error_rate in alerts payload: %+v", alerts)
	}

	monitorHTML := mustGetText(t, server.URL+"/monitor")
	for _, fragment := range []string{"BigClaw Monitor", "Requests", "Error Rate", "Auto refresh every 5s"} {
		if !strings.Contains(monitorHTML, fragment) {
			t.Fatalf("expected %q in monitor html: %s", fragment, monitorHTML)
		}
	}

	if got := monitoring.RequestTotal(); got < 6 {
		t.Fatalf("expected request total >= 6, got %d", got)
	}
}

func mustGetText(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}
	return string(body)
}

func mustGetJSON(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get %s: %v", url, err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func timeNow() time.Time {
	return time.Now()
}
