package executor

import (
	"context"
	"fmt"
	"io"
	"maps"
	"regexp"
	"sort"
	"strings"
	"time"

	"bigclaw-go/internal/domain"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubernetes "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type KubernetesConfig struct {
	Namespace           string
	Image               string
	ServiceAccountName  string
	KubeconfigPath      string
	PollInterval        time.Duration
	CleanupFinishedJobs bool
	BackoffLimit        int32
	JobTTLSeconds       int32
	LogTailLines        int64
}

type KubernetesAPI interface {
	CreateJob(context.Context, string, *batchv1.Job) (*batchv1.Job, error)
	GetJob(context.Context, string, string) (*batchv1.Job, error)
	DeleteJob(context.Context, string, string) error
	ListPods(context.Context, string, map[string]string) ([]corev1.Pod, error)
	GetPodLogs(context.Context, string, string, *corev1.PodLogOptions) (string, error)
}

type KubernetesRunner struct {
	api KubernetesAPI
	cfg KubernetesConfig
	now func() time.Time
}

func NewKubernetesRunner(cfg KubernetesConfig) (*KubernetesRunner, error) {
	api, err := newKubernetesClient(cfg)
	if err != nil {
		return nil, err
	}
	return NewKubernetesRunnerWithAPI(cfg, api), nil
}

func NewKubernetesRunnerWithAPI(cfg KubernetesConfig, api KubernetesAPI) *KubernetesRunner {
	if cfg.Namespace == "" {
		cfg.Namespace = "default"
	}
	if cfg.Image == "" {
		cfg.Image = "alpine:3.20"
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.JobTTLSeconds <= 0 {
		cfg.JobTTLSeconds = 300
	}
	if cfg.LogTailLines <= 0 {
		cfg.LogTailLines = 200
	}
	return &KubernetesRunner{api: api, cfg: cfg, now: time.Now}
}

func (r *KubernetesRunner) Kind() domain.ExecutorKind { return domain.ExecutorKubernetes }

func (r *KubernetesRunner) Capability() Capability {
	return Capability{Kind: domain.ExecutorKubernetes, MaxConcurrency: 256, SupportsShell: true}
}

func (r *KubernetesRunner) Execute(ctx context.Context, task domain.Task) Result {
	job := r.buildJob(task)
	created, err := r.api.CreateJob(ctx, r.cfg.Namespace, job)
	if err != nil {
		return Result{ShouldRetry: true, Message: fmt.Sprintf("create kubernetes job: %v", err), FinishedAt: r.now()}
	}

	artifacts := []string{fmt.Sprintf("k8s://jobs/%s/%s", r.cfg.Namespace, created.Name)}
	ticker := time.NewTicker(r.cfg.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = r.api.DeleteJob(context.Background(), r.cfg.Namespace, created.Name)
			return Result{ShouldRetry: true, Message: ctx.Err().Error(), Artifacts: artifacts, FinishedAt: r.now()}
		case <-ticker.C:
			current, err := r.api.GetJob(ctx, r.cfg.Namespace, created.Name)
			if err != nil {
				if apierrors.IsNotFound(err) {
					return Result{ShouldRetry: true, Message: "kubernetes job disappeared", Artifacts: artifacts, FinishedAt: r.now()}
				}
				return Result{ShouldRetry: true, Message: fmt.Sprintf("get kubernetes job: %v", err), Artifacts: artifacts, FinishedAt: r.now()}
			}
			if current.Status.Succeeded > 0 {
				logs, podArtifacts := r.collectLogs(ctx, created.Name)
				artifacts = append(artifacts, podArtifacts...)
				if r.cfg.CleanupFinishedJobs {
					_ = r.api.DeleteJob(context.Background(), r.cfg.Namespace, created.Name)
				}
				message := fmt.Sprintf("kubernetes job %s succeeded", created.Name)
				if logs != "" {
					message += ": " + trimMessage(logs)
				}
				return Result{Success: true, Message: message, Artifacts: artifacts, FinishedAt: r.now()}
			}
			if current.Status.Failed > 0 {
				logs, podArtifacts := r.collectLogs(ctx, created.Name)
				artifacts = append(artifacts, podArtifacts...)
				if r.cfg.CleanupFinishedJobs {
					_ = r.api.DeleteJob(context.Background(), r.cfg.Namespace, created.Name)
				}
				message := fmt.Sprintf("kubernetes job %s failed", created.Name)
				if logs != "" {
					message += ": " + trimMessage(logs)
				}
				return Result{DeadLetter: true, Message: message, Artifacts: artifacts, FinishedAt: r.now()}
			}
		}
	}
}

func (r *KubernetesRunner) buildJob(task domain.Task) *batchv1.Job {
	labels := map[string]string{
		"app.kubernetes.io/name": "bigclaw",
		"bigclaw/task-id":        sanitizeKubernetesName(task.ID),
	}
	for key, value := range task.Metadata {
		labels[sanitizeLabelKey(key)] = sanitizeLabelValue(value)
	}
	command, args := resolveKubernetesCommand(task)
	envVars := make([]corev1.EnvVar, 0, len(task.Environment))
	keys := make([]string, 0, len(task.Environment))
	for key := range task.Environment {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		envVars = append(envVars, corev1.EnvVar{Name: key, Value: task.Environment[key]})
	}
	backoffLimit := r.cfg.BackoffLimit
	ttlSecondsAfterFinished := r.cfg.JobTTLSeconds
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName(task.ID, r.now()),
			Namespace: r.cfg.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: maps.Clone(labels)},
				Spec: corev1.PodSpec{
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: r.cfg.ServiceAccountName,
					Containers: []corev1.Container{{
						Name:            "executor",
						Image:           resolveContainerImage(task, r.cfg),
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         command,
						Args:            args,
						Env:             envVars,
						WorkingDir:      task.WorkingDir,
					}},
				},
			},
		},
	}
	if task.ExecutionTimeoutSeconds > 0 {
		deadline := task.ExecutionTimeoutSeconds
		job.Spec.ActiveDeadlineSeconds = &deadline
	}
	return job
}

func (r *KubernetesRunner) collectLogs(ctx context.Context, jobName string) (string, []string) {
	pods, err := r.api.ListPods(ctx, r.cfg.Namespace, map[string]string{"job-name": jobName})
	if err != nil || len(pods) == 0 {
		return "", nil
	}
	sort.SliceStable(pods, func(i, j int) bool { return pods[i].Name < pods[j].Name })
	artifacts := make([]string, 0, len(pods))
	for _, pod := range pods {
		artifacts = append(artifacts, fmt.Sprintf("k8s://pods/%s/%s", r.cfg.Namespace, pod.Name))
	}
	logs, err := r.api.GetPodLogs(ctx, r.cfg.Namespace, pods[0].Name, &corev1.PodLogOptions{TailLines: &r.cfg.LogTailLines})
	if err != nil {
		return "", artifacts
	}
	return logs, artifacts
}

type kubernetesClient struct{ client kubernetes.Interface }

func newKubernetesClient(cfg KubernetesConfig) (KubernetesAPI, error) {
	restConfig, err := buildKubernetesRESTConfig(cfg.KubeconfigPath)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}
	return &kubernetesClient{client: client}, nil
}

func buildKubernetesRESTConfig(kubeconfigPath string) (*rest.Config, error) {
	if kubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	}
	if restConfig, err := rest.InClusterConfig(); err == nil {
		return restConfig, nil
	}
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{}).ClientConfig()
}

func (k *kubernetesClient) CreateJob(ctx context.Context, namespace string, job *batchv1.Job) (*batchv1.Job, error) {
	return k.client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
}

func (k *kubernetesClient) GetJob(ctx context.Context, namespace, name string) (*batchv1.Job, error) {
	return k.client.BatchV1().Jobs(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (k *kubernetesClient) DeleteJob(ctx context.Context, namespace, name string) error {
	policy := metav1.DeletePropagationBackground
	return k.client.BatchV1().Jobs(namespace).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &policy})
}

func (k *kubernetesClient) ListPods(ctx context.Context, namespace string, selector map[string]string) ([]corev1.Pod, error) {
	listOptions := metav1.ListOptions{LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: selector})}
	pods, err := k.client.CoreV1().Pods(namespace).List(ctx, listOptions)
	if err != nil {
		return nil, err
	}
	return pods.Items, nil
}

func (k *kubernetesClient) GetPodLogs(ctx context.Context, namespace, name string, opts *corev1.PodLogOptions) (string, error) {
	stream, err := k.client.CoreV1().Pods(namespace).GetLogs(name, opts).Stream(ctx)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	contents, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func resolveContainerImage(task domain.Task, cfg KubernetesConfig) string {
	if task.ContainerImage != "" {
		return task.ContainerImage
	}
	if cfg.Image != "" {
		return cfg.Image
	}
	return "alpine:3.20"
}

func resolveKubernetesCommand(task domain.Task) ([]string, []string) {
	if len(task.Command) > 0 {
		return task.Command, task.Args
	}
	if task.Entrypoint != "" {
		return []string{"/bin/sh", "-lc", task.Entrypoint}, nil
	}
	message := fmt.Sprintf("echo 'BigClaw task %s: %s'", task.ID, escapeSingleQuotes(task.Title))
	return []string{"/bin/sh", "-lc", message}, nil
}

var invalidKubernetesName = regexp.MustCompile(`[^a-z0-9-]+`)

func jobName(taskID string, now time.Time) string {
	base := sanitizeKubernetesName(taskID)
	if base == "" {
		base = "task"
	}
	name := fmt.Sprintf("bigclaw-%s-%d", base, now.Unix())
	if len(name) > 63 {
		name = name[:63]
	}
	return strings.Trim(name, "-")
}

func sanitizeKubernetesName(input string) string {
	cleaned := strings.ToLower(input)
	cleaned = invalidKubernetesName.ReplaceAllString(cleaned, "-")
	cleaned = strings.Trim(cleaned, "-")
	if len(cleaned) > 40 {
		cleaned = cleaned[:40]
	}
	return strings.Trim(cleaned, "-")
}

func sanitizeLabelKey(input string) string {
	cleaned := sanitizeKubernetesName(strings.ReplaceAll(input, "/", "-"))
	if cleaned == "" {
		return "bigclaw-metadata"
	}
	return cleaned
}

func sanitizeLabelValue(input string) string {
	cleaned := sanitizeKubernetesName(input)
	if cleaned == "" {
		return "unknown"
	}
	return cleaned
}

func escapeSingleQuotes(input string) string {
	return strings.ReplaceAll(input, "'", "'\"'\"'")
}

func trimMessage(input string) string {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) > 200 {
		return trimmed[:200]
	}
	return trimmed
}
