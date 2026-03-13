package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"bigclaw-go/internal/domain"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

type fakeKubernetesAPI struct {
	createdJobs []*batchv1.Job
	job         *batchv1.Job
	pods        []corev1.Pod
	logs        string
}

func (f *fakeKubernetesAPI) CreateJob(_ context.Context, _ string, job *batchv1.Job) (*batchv1.Job, error) {
	copied := job.DeepCopy()
	f.createdJobs = append(f.createdJobs, copied)
	f.job = copied.DeepCopy()
	return copied, nil
}

func (f *fakeKubernetesAPI) GetJob(_ context.Context, _ string, _ string) (*batchv1.Job, error) {
	if f.job == nil {
		return nil, errors.New("job not found")
	}
	f.job.Status.Succeeded = 1
	return f.job.DeepCopy(), nil
}

func (f *fakeKubernetesAPI) DeleteJob(context.Context, string, string) error { return nil }

func (f *fakeKubernetesAPI) ListPods(context.Context, string, map[string]string) ([]corev1.Pod, error) {
	return f.pods, nil
}

func (f *fakeKubernetesAPI) GetPodLogs(context.Context, string, string, *corev1.PodLogOptions) (string, error) {
	return f.logs, nil
}

func TestKubernetesRunnerExecuteCreatesRealJobSpec(t *testing.T) {
	api := &fakeKubernetesAPI{
		pods: []corev1.Pod{{}},
		logs: "job finished successfully",
	}
	runner := NewKubernetesRunnerWithAPI(KubernetesConfig{Namespace: "bigclaw", Image: "busybox:1.36", PollInterval: time.Millisecond}, api)
	runner.now = func() time.Time { return time.Unix(1700000000, 0) }

	result := runner.Execute(context.Background(), domain.Task{
		ID:             "OPE-181",
		Title:          "run on kubernetes",
		ContainerImage: "bash:5.2",
		Entrypoint:     "echo hello from kubernetes",
		Environment:    map[string]string{"BIGCLAW_TASK_ID": "OPE-181"},
		Metadata:       map[string]string{"tenant": "openagi"},
	})

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if len(api.createdJobs) != 1 {
		t.Fatalf("expected one created job, got %d", len(api.createdJobs))
	}
	job := api.createdJobs[0]
	if got := job.Spec.Template.Spec.Containers[0].Image; got != "bash:5.2" {
		t.Fatalf("expected task image override, got %s", got)
	}
	if got := job.Spec.Template.Spec.Containers[0].Command; len(got) != 3 || got[2] != "echo hello from kubernetes" {
		t.Fatalf("unexpected command: %#v", got)
	}
	if !strings.Contains(result.Message, "kubernetes job") {
		t.Fatalf("expected kubernetes success message, got %s", result.Message)
	}
}
