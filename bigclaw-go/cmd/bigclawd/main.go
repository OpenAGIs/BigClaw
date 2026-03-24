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
	eventPlanBackend := cfg.EventLogBackend
	switch {
	case cfg.EventLogRemoteURL != "":
		eventPlanBackend = "http"
		eventLog, err = events.NewHTTPEventLog(cfg.EventLogRemoteURL, cfg.EventLogRemoteBearer)
		if err != nil {
			panic(err)
		}
	case cfg.EventLogSQLitePath != "":
		eventPlanBackend = "sqlite"
		eventLog, err = events.NewSQLiteEventLogWithOptions(cfg.EventLogSQLitePath, events.SQLiteEventLogOptions{Retention: cfg.EventRetention})
		if err != nil {
			panic(err)
		}
		defer closeEventLog(eventLog)
	default:
		eventLog, err = buildEventLog(cfg)
		if err != nil {
			panic(err)
		}
		if eventLog != nil {
			eventPlanBackend = eventLog.Backend()
		}
	}

	bus := events.NewBus()
	if eventLog != nil {
		bus.AddSink(eventLog)
		bus.SetCapabilities(eventLog.Capabilities())
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
	subscriberLeases, err := buildSubscriberLeaseStore(cfg)
	if err != nil {
		panic(err)
	}
	defer closeSubscriberLeaseStore(subscriberLeases)
	schedulerRuntime := scheduler.NewWithStores(policyStore, fairnessStore)
	pool := buildWorkerPool(cfg, q, schedulerRuntime, registry, bus, recorder, controller)
	loop := &orchestrator.Loop{Runtime: pool, Quota: scheduler.QuotaSnapshot{ConcurrentLimit: cfg.MaxConcurrentRuns, BudgetRemaining: cfg.DefaultBudgetCents}, PollInterval: cfg.PollInterval}

	if cfg.BootstrapTasks {
		seed(context.Background(), q)
	}
	server := &api.Server{
		Recorder:         recorder,
		Queue:            q,
		Executors:        registry.Kinds(),
		Bus:              bus,
		EventPlan:        events.NewDurabilityPlanWithBrokerConfig(eventPlanBackend, cfg.EventLogTargetBackend, cfg.EventLogReplicationFactor, brokerRuntimeConfig(cfg)),
		EventLog:         eventLog,
		SubscriberLeases: subscriberLeases,
		Worker:           pool,
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

func buildSubscriberLeaseStore(cfg config.Config) (events.SubscriberLeaseStore, error) {
	if cfg.SubscriberLeaseSQLitePath == "" {
		return events.NewSubscriberLeaseCoordinator(), nil
	}
	return events.NewSQLiteSubscriberLeaseStore(cfg.SubscriberLeaseSQLitePath)
}

func closeSubscriberLeaseStore(store events.SubscriberLeaseStore) {
	type closer interface{ Close() error }
	if closable, ok := store.(closer); ok {
		_ = closable.Close()
	}
}

func buildEventLog(cfg config.Config) (events.EventLog, error) {
	switch cfg.EventLogBackend {
	case "", string(events.EventLogBackendMemory):
		return nil, nil
	case string(events.EventLogBackendBroker):
		broker := brokerRuntimeConfig(cfg)
		if err := broker.Validate(); err != nil {
			return nil, err
		}
		if broker.Driver == events.BrokerDriverStub {
			return events.NewBrokerStubEventLog(), nil
		}
		return nil, fmt.Errorf("event log backend %q is not implemented yet; driver=%s topic=%s contract validated for the future adapter", cfg.EventLogBackend, broker.Driver, broker.Topic)
	default:
		return nil, fmt.Errorf("unsupported event log backend: %s", cfg.EventLogBackend)
	}
}

func brokerRuntimeConfig(cfg config.Config) events.BrokerRuntimeConfig {
	return events.BrokerRuntimeConfig{
		Driver:             cfg.EventLogBrokerDriver,
		URLs:               cfg.EventLogBrokerURLs,
		Topic:              cfg.EventLogBrokerTopic,
		ConsumerGroup:      cfg.EventLogConsumerGroup,
		PublishTimeout:     cfg.EventLogPublishTimeout,
		ReplayLimit:        cfg.EventLogReplayLimit,
		CheckpointInterval: cfg.EventLogCheckpointInterval,
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

func buildWorkerPool(
	cfg config.Config,
	q queue.Queue,
	schedulerRuntime *scheduler.Scheduler,
	registry *executor.Registry,
	bus *events.Bus,
	recorder *observability.Recorder,
	controller *control.Controller,
) *worker.Pool {
	workerCount := cfg.MaxConcurrentRuns
	if workerCount < 1 {
		workerCount = 1
	}

	runtimes := make([]*worker.Runtime, 0, workerCount)
	for index := 0; index < workerCount; index++ {
		runtimes = append(runtimes, &worker.Runtime{
			WorkerID:    fmt.Sprintf("worker-%d", index+1),
			NodeID:      cfg.NodeID,
			HostProfile: cfg.HostProfile,
			PoolID:      cfg.CapacityPool,
			Queue:       q,
			Scheduler:   schedulerRuntime,
			Registry:    registry,
			Bus:         bus,
			Recorder:    recorder,
			Control:     controller,
			LeaseTTL:    cfg.LeaseTTL,
			TaskTimeout: cfg.TaskTimeout,
		})
	}

	return worker.NewPool(runtimes...)
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
