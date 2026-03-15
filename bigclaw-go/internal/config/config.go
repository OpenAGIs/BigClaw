package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	QueueBackend                  string
	EventWebhookURLs              []string
	EventWebhookBearerToken       string
	EventWebhookTimeout           time.Duration
	QueueSQLitePath               string
	AuditLogPath                  string
	ServiceName                   string
	LeaseTTL                      time.Duration
	TaskTimeout                   time.Duration
	PollInterval                  time.Duration
	MaxConcurrentRuns             int
	DefaultBudgetCents            int64
	DefaultExecutor               string
	QueueFilePath                 string
	HTTPAddr                      string
	BootstrapTasks                bool
	KubernetesNamespace           string
	KubernetesImage               string
	KubernetesServiceAccount      string
	KubernetesKubeconfigPath      string
	KubernetesPollInterval        time.Duration
	KubernetesCleanupFinishedJobs bool
	KubernetesBackoffLimit        int32
	KubernetesJobTTLSeconds       int32
	KubernetesLogTailLines        int64
	RayAddress                    string
	RayPollInterval               time.Duration
	RayHTTPTimeout                time.Duration
	RayBearerToken                string
	SchedulerPolicyPath           string
	SchedulerPolicySQLitePath     string
	SchedulerFairnessSQLitePath   string
}

func Default() Config {
	return Config{
		QueueBackend:                  "file",
		EventWebhookTimeout:           5 * time.Second,
		QueueSQLitePath:               "./state/queue.db",
		AuditLogPath:                  "./state/audit.jsonl",
		ServiceName:                   "bigclawd",
		LeaseTTL:                      2 * time.Minute,
		TaskTimeout:                   30 * time.Second,
		PollInterval:                  100 * time.Millisecond,
		MaxConcurrentRuns:             32,
		DefaultBudgetCents:            1000,
		DefaultExecutor:               "local",
		QueueFilePath:                 "./state/queue.json",
		HTTPAddr:                      ":8080",
		BootstrapTasks:                false,
		KubernetesNamespace:           "default",
		KubernetesImage:               "alpine:3.20",
		KubernetesPollInterval:        2 * time.Second,
		KubernetesCleanupFinishedJobs: false,
		KubernetesBackoffLimit:        0,
		KubernetesJobTTLSeconds:       300,
		KubernetesLogTailLines:        200,
		RayAddress:                    "http://127.0.0.1:8265",
		RayPollInterval:               time.Second,
		RayHTTPTimeout:                10 * time.Second,
		SchedulerPolicyPath:           "",
		SchedulerPolicySQLitePath:     "",
		SchedulerFairnessSQLitePath:   "",
	}
}

func LoadFromEnv() Config {
	cfg := Default()
	cfg.QueueBackend = getString("BIGCLAW_QUEUE_BACKEND", cfg.QueueBackend)
	cfg.EventWebhookURLs = splitCSV(getString("BIGCLAW_EVENT_WEBHOOK_URLS", ""))
	cfg.EventWebhookBearerToken = getString("BIGCLAW_EVENT_WEBHOOK_BEARER_TOKEN", cfg.EventWebhookBearerToken)
	cfg.EventWebhookTimeout = getDuration("BIGCLAW_EVENT_WEBHOOK_TIMEOUT", cfg.EventWebhookTimeout)
	cfg.QueueSQLitePath = getString("BIGCLAW_QUEUE_SQLITE_PATH", cfg.QueueSQLitePath)
	cfg.AuditLogPath = getString("BIGCLAW_AUDIT_LOG_PATH", cfg.AuditLogPath)
	cfg.ServiceName = getString("BIGCLAW_SERVICE_NAME", cfg.ServiceName)
	cfg.QueueFilePath = getString("BIGCLAW_QUEUE_FILE", cfg.QueueFilePath)
	cfg.HTTPAddr = getString("BIGCLAW_HTTP_ADDR", cfg.HTTPAddr)
	cfg.BootstrapTasks = getBool("BIGCLAW_BOOTSTRAP_TASKS", cfg.BootstrapTasks)
	cfg.KubernetesNamespace = getString("BIGCLAW_KUBERNETES_NAMESPACE", cfg.KubernetesNamespace)
	cfg.KubernetesImage = getString("BIGCLAW_KUBERNETES_IMAGE", cfg.KubernetesImage)
	cfg.KubernetesServiceAccount = getString("BIGCLAW_KUBERNETES_SERVICE_ACCOUNT", cfg.KubernetesServiceAccount)
	cfg.KubernetesKubeconfigPath = getString("BIGCLAW_KUBECONFIG", getString("KUBECONFIG", cfg.KubernetesKubeconfigPath))
	cfg.RayAddress = getString("BIGCLAW_RAY_ADDRESS", getString("RAY_ADDRESS", cfg.RayAddress))
	cfg.RayBearerToken = getString("BIGCLAW_RAY_BEARER_TOKEN", cfg.RayBearerToken)
	cfg.SchedulerPolicyPath = getString("BIGCLAW_SCHEDULER_POLICY_PATH", cfg.SchedulerPolicyPath)
	cfg.SchedulerPolicySQLitePath = getString("BIGCLAW_SCHEDULER_POLICY_SQLITE_PATH", cfg.SchedulerPolicySQLitePath)
	cfg.SchedulerFairnessSQLitePath = getString("BIGCLAW_SCHEDULER_FAIRNESS_SQLITE_PATH", cfg.SchedulerFairnessSQLitePath)
	cfg.LeaseTTL = getDuration("BIGCLAW_LEASE_TTL", cfg.LeaseTTL)
	cfg.TaskTimeout = getDuration("BIGCLAW_TASK_TIMEOUT", cfg.TaskTimeout)
	cfg.PollInterval = getDuration("BIGCLAW_POLL_INTERVAL", cfg.PollInterval)
	cfg.KubernetesPollInterval = getDuration("BIGCLAW_KUBERNETES_POLL_INTERVAL", cfg.KubernetesPollInterval)
	cfg.RayPollInterval = getDuration("BIGCLAW_RAY_POLL_INTERVAL", cfg.RayPollInterval)
	cfg.RayHTTPTimeout = getDuration("BIGCLAW_RAY_HTTP_TIMEOUT", cfg.RayHTTPTimeout)
	cfg.MaxConcurrentRuns = getInt("BIGCLAW_MAX_CONCURRENT_RUNS", cfg.MaxConcurrentRuns)
	cfg.DefaultBudgetCents = getInt64("BIGCLAW_DEFAULT_BUDGET_CENTS", cfg.DefaultBudgetCents)
	cfg.KubernetesCleanupFinishedJobs = getBool("BIGCLAW_KUBERNETES_CLEANUP", cfg.KubernetesCleanupFinishedJobs)
	cfg.KubernetesBackoffLimit = int32(getInt("BIGCLAW_KUBERNETES_BACKOFF_LIMIT", int(cfg.KubernetesBackoffLimit)))
	cfg.KubernetesJobTTLSeconds = int32(getInt("BIGCLAW_KUBERNETES_JOB_TTL_SECONDS", int(cfg.KubernetesJobTTLSeconds)))
	cfg.KubernetesLogTailLines = getInt64("BIGCLAW_KUBERNETES_LOG_TAIL_LINES", cfg.KubernetesLogTailLines)
	return cfg
}

func getString(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func getInt64(key string, fallback int64) int64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return fallback
}

func splitCSV(value string) []string {
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
