package executor

import (
	"context"
	"time"

	"bigclaw-go/internal/domain"
)

type LocalRunner struct{}

func (LocalRunner) Kind() domain.ExecutorKind { return domain.ExecutorLocal }

func (LocalRunner) Capability() Capability {
	return Capability{Kind: domain.ExecutorLocal, MaxConcurrency: 64, SupportsShell: true}
}

func (LocalRunner) Execute(ctx context.Context, task domain.Task) Result {
	select {
	case <-ctx.Done():
		return Result{ShouldRetry: true, Message: ctx.Err().Error(), FinishedAt: time.Now()}
	case <-time.After(10 * time.Millisecond):
		return Result{Success: true, Message: "local execution completed", FinishedAt: time.Now(), Artifacts: []string{"stdout.log"}}
	}
}
