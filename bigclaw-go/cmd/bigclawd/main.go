package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bigclaw-go/internal/api"
	"bigclaw-go/internal/config"
	"bigclaw-go/internal/control"
	"bigclaw-go/internal/domain"
	"bigclaw-go/internal/events"
	"bigclaw-go/internal/executor"
	"bigclaw-go/internal/observability"
	"bigclaw-go/internal/orchestrator"
	"bigclaw-go/internal/queue"
	"bigclaw-go/internal/scheduler"
	"bigclaw-go/internal/worker"
)

func main() {
	cfg := config.LoadFromEnv()
	if err := validateEventBackend(cfg); err != nil {
		panic(err)
	}
	q, err := buildQueue(cfg)
	if err != nil {
		panic(err)
	}
	defer closeQueue(q)

	var eventLog events.EventLog
	var eventLogCapability *events.BackendCapabilities
	switch {
	case cfg.EventLogRemoteURL != "":
		eventLog, err = events.NewHTTPEventLog(cfg.EventLogRemoteURL, cfg.EventLogRemoteBearer)
		if err != nil {
			panic(err)
		}
		capability := eventLog.Capabilities()
		eventLogCapability = &capability
	case cfg.EventLogSQLitePath != "":
		eventLog, err = events.NewSQLiteEventLog(cfg.EventLogSQLitePath)
		if err != nil {
			panic(err)
		}
		capability := eventLog.Capabilities()
		eventLogCapability = &capability
		defer closeEventLog(eventLog)
	default:
		eventLogCapability, err = buildEventLogCapabilities(cfg)
		if err != nil {
			panic(err)
		}
	}

	bus := events.NewBus()
	if eventLogCapability != nil {
		bus.SetCapabilities(*eventLogCapability)
	}
	if eventLog != nil {
		bus.AddSink(eventLog)
	}
	recorder := buildRecorder(cfg)
	bus.AddSink(events.RecorderSink{Recorder: recorder})
	if len(cfg.EventWebhookURLs) > 0 {
		bus.AddSink(events.NewWebhookSink(events.WebhookConfig{
			URLs:        cfg.EventWebhookURLs,
			BearerToken: cfg.EventWebhookBearerToken,
			Timeout:     cfg.EventWebhookTimeout,
		}))
	}

	registry := buildRegistry(cfg)
	policyStore, err := scheduler.NewPolicyStoreWithSQLite(cfg.SchedulerPolicyPath, cfg.SchedulerPolicySQLitePath)
	if err != nil {
		panic(err)
	}
	defer closePolicyStore(policyStore)
	fairnessStore, err := scheduler.NewFairnessStoreWithRemote(cfg.SchedulerFairnessSQLitePath, cfg.SchedulerFairnessRemoteURL, cfg.SchedulerFairnessRemoteBearer)
	if err != nil {
		panic(err)
	}
	defer closeFairnessStore(fairnessStore)

	controller := control.New()
	subscriberLeases := events.NewSubscriberLeaseCoordinator()
	schedulerRuntime := scheduler.NewWithStores(policyStore, fairnessStore)
	runtime := &worker.Runtime{
		WorkerID:    "bootstrap-worker",
		Queue:       q,
		Scheduler:   schedulerRuntime,
		Registry:    registry,
		Bus:         bus,
		Recorder:    recorder,
		Control:     controller,
		LeaseTTL:    cfg.LeaseTTL,
		TaskTimeout: cfg.TaskTimeout,
	}
	loop := &orchestrator.Loop{Runtime: runtime, Quota: scheduler.QuotaSnapshot{ConcurrentLimit: cfg.MaxConcurrentRuns, BudgetRemaining: cfg.DefaultBudgetCents}, PollInterval: cfg.PollInterval}

	if cfg.BootstrapTasks {
		seed(context.Background(), q)
	}
	server := &api.Server{
		Recorder:         recorder,
		Queue:            q,
		Executors:        registry.Kinds(),
		Bus:              bus,
		EventPlan:        events.NewDurabilityPlan(cfg.EventLogBackend, cfg.EventLogTargetBackend, cfg.EventLogReplicationFactor),
		EventLog:         eventLog,
		SubscriberLeases: subscriberLeases,
		Worker:           runtime,
		Control:          controller,
		SchedulerPolicy:  policyStore,
		SchedulerRuntime: schedulerRuntime,
	}
	httpServer := &http.Server{Addr: cfg.HTTPAddr, Handler: server.Handler()}
	go func() {
		_ = httpServer.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go loop.Run(ctx)

	fmt.Printf("%s started queue=%T http=%s executors=%v\n", cfg.ServiceName, q, cfg.HTTPAddr, registry.Kinds())
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = httpServer.Shutdown(shutdownCtx)
	fmt.Printf("%s stopped events=%d\n", cfg.ServiceName, len(bus.Replay()))
}

func buildEventLogCapabilities(cfg config.Config) (*events.BackendCapabilities, error) {
	switch cfg.EventLogBackend {
	case "", string(events.EventLogBackendMemory):
		return nil, nil
	case string(events.EventLogBackendBroker):
		broker := events.BrokerRuntimeConfig{
			Driver:             cfg.EventLogBrokerDriver,
			URLs:               cfg.EventLogBrokerURLs,
			Topic:              cfg.EventLogBrokerTopic,
			ConsumerGroup:      cfg.EventLogConsumerGroup,
			PublishTimeout:     cfg.EventLogPublishTimeout,
			ReplayLimit:        cfg.EventLogReplayLimit,
			CheckpointInterval: cfg.EventLogCheckpointInterval,
		}
		if err := broker.Validate(); err != nil {
			return nil, err
		}
		capability := events.BrokerBootstrapCapabilities(broker)
		return &capability, nil
	default:
		return nil, fmt.Errorf("unsupported event log backend: %s", cfg.EventLogBackend)
	}
}

func buildQueue(cfg config.Config) (queue.Queue, error) {
	switch cfg.QueueBackend {
	case "sqlite":
		return queue.NewSQLiteQueue(cfg.QueueSQLitePath)
	case "file", "":
		return queue.NewFileQueue(cfg.QueueFilePath)
	default:
		return nil, fmt.Errorf("unsupported queue backend: %s", cfg.QueueBackend)
	}
}

func closeQueue(q queue.Queue) {
	type closer interface{ Close() error }
	if closerQueue, ok := q.(closer); ok {
		_ = closerQueue.Close()
	}
}

func closeEventLog(store events.EventLog) {
	if store != nil {
		_ = store.Close()
	}
}

func closePolicyStore(store *scheduler.PolicyStore) {
	if store != nil {
		_ = store.Close()
	}
}

func closeFairnessStore(store scheduler.FairnessStore) {
	type closer interface{ Close() error }
	if closable, ok := store.(closer); ok {
		_ = closable.Close()
	}
}

func buildRecorder(cfg config.Config) *observability.Recorder {
	if cfg.AuditLogPath == "" {
		return observability.NewRecorder()
	}
	sink, err := observability.NewJSONLAuditSink(cfg.AuditLogPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "audit sink disabled: %v\n", err)
		return observability.NewRecorder()
	}
	return observability.NewRecorderWithSinks(sink)
}

func buildRegistry(cfg config.Config) *executor.Registry {
	runners := []executor.Runner{executor.LocalRunner{}}
	kubernetesRunner, err := executor.NewKubernetesRunner(executor.KubernetesConfig{
		Namespace:           cfg.KubernetesNamespace,
		Image:               cfg.KubernetesImage,
		ServiceAccountName:  cfg.KubernetesServiceAccount,
		KubeconfigPath:      cfg.KubernetesKubeconfigPath,
		PollInterval:        cfg.KubernetesPollInterval,
		CleanupFinishedJobs: cfg.KubernetesCleanupFinishedJobs,
		BackoffLimit:        cfg.KubernetesBackoffLimit,
		JobTTLSeconds:       cfg.KubernetesJobTTLSeconds,
		LogTailLines:        cfg.KubernetesLogTailLines,
	})
	if err == nil {
		runners = append(runners, kubernetesRunner)
	} else {
		fmt.Fprintf(os.Stderr, "kubernetes runner disabled: %v\n", err)
	}
	rayRunner, err := executor.NewRayRunner(executor.RayConfig{
		Address:      cfg.RayAddress,
		PollInterval: cfg.RayPollInterval,
		HTTPTimeout:  cfg.RayHTTPTimeout,
		BearerToken:  cfg.RayBearerToken,
	})
	if err == nil {
		runners = append(runners, rayRunner)
	} else {
		fmt.Fprintf(os.Stderr, "ray runner disabled: %v\n", err)
	}
	return executor.NewRegistry(runners...)
}

func seed(ctx context.Context, q queue.Queue) {
	_ = q.Enqueue(ctx, domain.Task{ID: "bootstrap-local", Title: "bootstrap local task", Priority: 1, RiskLevel: domain.RiskLow, BudgetCents: 100, Entrypoint: "echo hello from local", CreatedAt: time.Now(), UpdatedAt: time.Now()})
}

func validateEventBackend(cfg config.Config) error {
	report := events.ValidateBackendConfig(events.BackendConfig{
		Backend:           events.BackendKind(cfg.EventBackend),
		LogDSN:            cfg.EventLogDSN,
		CheckpointDSN:     cfg.EventCheckpointDSN,
		Retention:         cfg.EventRetention,
		RequireReplay:     cfg.EventRequireReplay,
		RequireCheckpoint: cfg.EventRequireCheckpoint,
		RequireFiltering:  cfg.EventRequireFiltering,
	})
	return report.Error()
}
