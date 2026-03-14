package queue

import (
	"context"
	"errors"
	"time"

	"bigclaw-go/internal/domain"
)

var ErrTaskNotFound = errors.New("task not found")
var ErrLeaseNotOwned = errors.New("lease not owned by worker")

type Lease struct {
	TaskID     string
	WorkerID   string
	ExpiresAt  time.Time
	Attempt    int
	AcquiredAt time.Time
}

type TaskSnapshot struct {
	Task         domain.Task `json:"task"`
	AvailableAt  time.Time   `json:"available_at,omitempty"`
	Attempt      int         `json:"attempt"`
	Leased       bool        `json:"leased"`
	LeaseWorker  string      `json:"lease_worker,omitempty"`
	LeaseExpires time.Time   `json:"lease_expires,omitempty"`
}

type Queue interface {
	Enqueue(context.Context, domain.Task) error
	LeaseNext(context.Context, string, time.Duration) (*domain.Task, *Lease, error)
	RenewLease(context.Context, *Lease, time.Duration) error
	Ack(context.Context, *Lease) error
	Requeue(context.Context, *Lease, time.Time) error
	DeadLetter(context.Context, *Lease, string) error
	ListDeadLetters(context.Context, int) ([]domain.Task, error)
	ReplayDeadLetter(context.Context, string) error
	Size(context.Context) int
}

type TaskInspector interface {
	GetTask(context.Context, string) (TaskSnapshot, error)
	ListTasks(context.Context, int) ([]TaskSnapshot, error)
}

type TaskController interface {
	CancelTask(context.Context, string, string) (TaskSnapshot, error)
}
