package api

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
)

type metricsSnapshot struct {
	QueueSize              int
	Events                 map[domain.EventType]int
	TraceCount             int
	RegisteredExecutors    []string
	WorkerPool             *workerPoolSummary
	Control                control.Snapshot
	EventDurability        any
	EventDurabilityRollout any
	EventLog               any
	RetentionWatermark     any
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot := s.buildMetricsSnapshot()
	if wantsPrometheusMetrics(r) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(renderPrometheusMetrics(snapshot)))
		return
	}
	writeJSON(w, http.StatusOK, metricsJSONPayload(snapshot))
}

func (s *Server) buildMetricsSnapshot() metricsSnapshot {
	snapshot := metricsSnapshot{
		QueueSize:              s.Queue.Size(context.Background()),
		Events:                 s.Recorder.Snapshot(),
		TraceCount:             len(s.Recorder.TraceSummaries(0)),
		RegisteredExecutors:    s.executorNames(),
		WorkerPool:             s.workerPoolSummary(),
		EventDurability:        s.EventPlan,
		EventDurabilityRollout: s.EventPlan.RolloutScorecard(),
		EventLog:               s.eventLogCapabilities(context.Background()),
		RetentionWatermark:     s.retentionWatermark(),
	}
	if s.Control != nil {
		snapshot.Control = s.Control.Snapshot()
	}
	if s.EventLog != nil {
		snapshot.EventLog = map[string]any{"backend": s.EventLog.Backend(), "capabilities": s.EventLog.Capabilities()}
	}
	return snapshot
}

func metricsJSONPayload(snapshot metricsSnapshot) map[string]any {
	return map[string]any{
		"queue_size":               snapshot.QueueSize,
		"events":                   snapshot.Events,
		"trace_count":              snapshot.TraceCount,
		"registered_executors":     snapshot.RegisteredExecutors,
		"event_durability":         snapshot.EventDurability,
		"event_durability_rollout": snapshot.EventDurabilityRollout,
		"event_log":                snapshot.EventLog,
		"retention_watermark":      snapshot.RetentionWatermark,
	}
}

func wantsPrometheusMetrics(r *http.Request) bool {
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	switch format {
	case "prometheus", "prom", "text":
		return true
	case "json", "":
	default:
		return false
	}
	accept := strings.ToLower(r.Header.Get("Accept"))
	return strings.Contains(accept, "text/plain") || strings.Contains(accept, "openmetrics-text")
}

func renderPrometheusMetrics(snapshot metricsSnapshot) string {
	var builder strings.Builder
	writePrometheusMetric(&builder, "bigclaw_queue_size", "gauge", "Current queue size.", nil, float64(snapshot.QueueSize))
	writePrometheusMetric(&builder, "bigclaw_trace_count", "gauge", "Number of in-memory traces.", nil, float64(snapshot.TraceCount))

	eventTypes := make([]string, 0, len(snapshot.Events))
	for eventType := range snapshot.Events {
		eventTypes = append(eventTypes, string(eventType))
	}
	sort.Strings(eventTypes)
	writePrometheusHeader(&builder, "bigclaw_events_total", "counter", "Recorded events by type.")
	for _, eventType := range eventTypes {
		builder.WriteString(prometheusSample("bigclaw_events_total", map[string]string{"event_type": eventType}, float64(snapshot.Events[domain.EventType(eventType)])))
	}

	writePrometheusHeader(&builder, "bigclaw_executor_registered", "gauge", "Registered executor availability.")
	for _, executor := range snapshot.RegisteredExecutors {
		builder.WriteString(prometheusSample("bigclaw_executor_registered", map[string]string{"executor": executor}, 1))
	}

	writePrometheusMetric(&builder, "bigclaw_control_paused", "gauge", "Whether the control plane is paused.", nil, boolGauge(snapshot.Control.Paused))
	writePrometheusMetric(&builder, "bigclaw_control_active_takeovers", "gauge", "Number of active human takeovers.", nil, float64(snapshot.Control.ActiveTakeovers))

	if snapshot.WorkerPool != nil {
		writePrometheusMetric(&builder, "bigclaw_worker_pool_total", "gauge", "Total workers in the pool.", nil, float64(snapshot.WorkerPool.TotalWorkers))
		writePrometheusMetric(&builder, "bigclaw_worker_pool_active", "gauge", "Active workers in the pool.", nil, float64(snapshot.WorkerPool.ActiveWorkers))
		writePrometheusMetric(&builder, "bigclaw_worker_pool_idle", "gauge", "Idle workers in the pool.", nil, float64(snapshot.WorkerPool.IdleWorkers))

		writePrometheusHeader(&builder, "bigclaw_worker_status", "gauge", "Worker state marker with executor labels.")
		writePrometheusHeader(&builder, "bigclaw_worker_successful_runs_total", "counter", "Successful runs per worker.")
		writePrometheusHeader(&builder, "bigclaw_worker_lease_renewals_total", "counter", "Lease renewals per worker.")
		writePrometheusHeader(&builder, "bigclaw_worker_retried_runs_total", "counter", "Retried runs per worker.")
		writePrometheusHeader(&builder, "bigclaw_worker_dead_letter_runs_total", "counter", "Dead-letter runs per worker.")
		writePrometheusHeader(&builder, "bigclaw_worker_cancelled_runs_total", "counter", "Cancelled runs per worker.")
		writePrometheusHeader(&builder, "bigclaw_worker_preemptions_total", "counter", "Preemptions issued per worker.")
		writePrometheusHeader(&builder, "bigclaw_worker_preemption_active", "gauge", "Whether the worker currently has an active preemption context.")
		for _, status := range snapshot.WorkerPool.Workers {
			executor := string(status.CurrentExecutor)
			if executor == "" {
				executor = "unassigned"
			}
			state := status.State
			if state == "" {
				state = "idle"
			}
			labels := map[string]string{
				"worker_id":        status.WorkerID,
				"state":            state,
				"current_executor": executor,
			}
			builder.WriteString(prometheusSample("bigclaw_worker_status", labels, 1))
			workerLabels := map[string]string{"worker_id": status.WorkerID}
			builder.WriteString(prometheusSample("bigclaw_worker_successful_runs_total", workerLabels, float64(status.SuccessfulRuns)))
			builder.WriteString(prometheusSample("bigclaw_worker_lease_renewals_total", workerLabels, float64(status.LeaseRenewals)))
			builder.WriteString(prometheusSample("bigclaw_worker_retried_runs_total", workerLabels, float64(status.RetriedRuns)))
			builder.WriteString(prometheusSample("bigclaw_worker_dead_letter_runs_total", workerLabels, float64(status.DeadLetterRuns)))
			builder.WriteString(prometheusSample("bigclaw_worker_cancelled_runs_total", workerLabels, float64(status.CancelledRuns)))
			builder.WriteString(prometheusSample("bigclaw_worker_preemptions_total", workerLabels, float64(status.PreemptionsIssued)))
			builder.WriteString(prometheusSample("bigclaw_worker_preemption_active", workerLabels, boolGauge(status.PreemptionActive)))
		}
	}

	return builder.String()
}

func writePrometheusMetric(builder *strings.Builder, name, metricType, help string, labels map[string]string, value float64) {
	writePrometheusHeader(builder, name, metricType, help)
	builder.WriteString(prometheusSample(name, labels, value))
}

func writePrometheusHeader(builder *strings.Builder, name, metricType, help string) {
	builder.WriteString("# HELP ")
	builder.WriteString(name)
	builder.WriteByte(' ')
	builder.WriteString(help)
	builder.WriteByte('\n')
	builder.WriteString("# TYPE ")
	builder.WriteString(name)
	builder.WriteByte(' ')
	builder.WriteString(metricType)
	builder.WriteByte('\n')
}

func prometheusSample(name string, labels map[string]string, value float64) string {
	var builder strings.Builder
	builder.WriteString(name)
	if len(labels) > 0 {
		keys := make([]string, 0, len(labels))
		for key := range labels {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		builder.WriteByte('{')
		for index, key := range keys {
			if index > 0 {
				builder.WriteByte(',')
			}
			builder.WriteString(key)
			builder.WriteString("=\"")
			builder.WriteString(escapePrometheusLabelValue(labels[key]))
			builder.WriteByte('"')
		}
		builder.WriteByte('}')
	}
	builder.WriteByte(' ')
	builder.WriteString(fmt.Sprintf("%g", value))
	builder.WriteByte('\n')
	return builder.String()
}

func escapePrometheusLabelValue(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "\n", "\\n", "\"", "\\\"")
	return replacer.Replace(value)
}

func boolGauge(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
