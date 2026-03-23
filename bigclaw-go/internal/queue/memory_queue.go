package queue

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"bigclaw-go/internal/domain"
)

type item struct {
	Task         domain.Task `json:"task"`
	AvailableAt  time.Time   `json:"available_at"`
	Attempt      int         `json:"attempt"`
	Leased       bool        `json:"leased"`
	LeaseWorker  string      `json:"lease_worker"`
	LeaseExpires time.Time   `json:"lease_expires"`
}

type MemoryQueue struct {
	mu    sync.Mutex
	items map[string]*item
}

func NewMemoryQueue() *MemoryQueue {
	return &MemoryQueue{items: make(map[string]*item)}
}

func (q *MemoryQueue) Enqueue(_ context.Context, task domain.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	if existing, ok := q.items[task.ID]; ok {
		existing.Task = task
		existing.AvailableAt = now
		existing.Leased = false
		existing.LeaseWorker = ""
		existing.LeaseExpires = time.Time{}
		existing.Task.State = domain.TaskQueued
		existing.Task.UpdatedAt = now
		return nil
	}

	task.State = domain.TaskQueued
	task.UpdatedAt = now
	q.items[task.ID] = &item{Task: task, AvailableAt: now}
	return nil
}

func (q *MemoryQueue) LeaseNext(_ context.Context, workerID string, ttl time.Duration) (*domain.Task, *Lease, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	q.recoverExpiredLeases(now)

	ordered := make([]*item, 0, len(q.items))
	for _, current := range q.items {
		if current.Leased || current.AvailableAt.After(now) || !isActionableState(current.Task.State) {
			continue
		}
		ordered = append(ordered, current)
	}
	if len(ordered) == 0 {
		return nil, nil, nil
	}

	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Task.Priority == ordered[j].Task.Priority {
			return ordered[i].Task.CreatedAt.Before(ordered[j].Task.CreatedAt)
		}
		return ordered[i].Task.Priority < ordered[j].Task.Priority
	})

	picked := ordered[0]
	picked.Leased = true
	picked.LeaseWorker = workerID
	picked.LeaseExpires = now.Add(ttl)
	picked.Task.State = domain.TaskLeased
	picked.Attempt++
	picked.Task.UpdatedAt = now

	lease := &Lease{
		TaskID:     picked.Task.ID,
		WorkerID:   workerID,
		ExpiresAt:  picked.LeaseExpires,
		Attempt:    picked.Attempt,
		AcquiredAt: now,
	}
	copy := picked.Task
	return &copy, lease, nil
}

func (q *MemoryQueue) RenewLease(_ context.Context, lease *Lease, ttl time.Duration) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[lease.TaskID]
	if !ok {
		return ErrTaskNotFound
	}
	if !current.Leased || current.LeaseWorker != lease.WorkerID || current.Attempt != lease.Attempt {
		return ErrLeaseNotOwned
	}
	if !current.LeaseExpires.After(time.Now()) {
		return ErrLeaseExpired
	}
	current.LeaseExpires = time.Now().Add(ttl)
	lease.ExpiresAt = current.LeaseExpires
	return nil
}

func (q *MemoryQueue) Ack(_ context.Context, lease *Lease) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[lease.TaskID]
	if !ok {
		return ErrTaskNotFound
	}
	if !current.Leased || current.LeaseWorker != lease.WorkerID || current.Attempt != lease.Attempt {
		return ErrLeaseNotOwned
	}
	if !current.LeaseExpires.After(time.Now()) {
		return ErrLeaseExpired
	}
	delete(q.items, lease.TaskID)
	return nil
}

func (q *MemoryQueue) Requeue(_ context.Context, lease *Lease, availableAt time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[lease.TaskID]
	if !ok {
		return ErrTaskNotFound
	}
	if !current.Leased || current.LeaseWorker != lease.WorkerID || current.Attempt != lease.Attempt {
		return ErrLeaseNotOwned
	}
	if !current.LeaseExpires.After(time.Now()) {
		return ErrLeaseExpired
	}
	if current.Task.State == domain.TaskCancelled || current.Task.State == domain.TaskDeadLetter {
		return ErrLeaseNotOwned
	}
	current.Leased = false
	current.LeaseWorker = ""
	current.LeaseExpires = time.Time{}
	current.AvailableAt = availableAt
	current.Task.State = domain.TaskQueued
	current.Task.UpdatedAt = time.Now()
	return nil
}

func (q *MemoryQueue) DeadLetter(_ context.Context, lease *Lease, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[lease.TaskID]
	if !ok {
		return ErrTaskNotFound
	}
	if !current.Leased || current.LeaseWorker != lease.WorkerID || current.Attempt != lease.Attempt {
		return ErrLeaseNotOwned
	}
	if !current.LeaseExpires.After(time.Now()) {
		return ErrLeaseExpired
	}
	if current.Task.State == domain.TaskCancelled || current.Task.State == domain.TaskDeadLetter {
		return ErrLeaseNotOwned
	}
	current.Leased = false
	current.LeaseWorker = ""
	current.LeaseExpires = time.Time{}
	current.AvailableAt = time.Now()
	current.Task.State = domain.TaskDeadLetter
	current.Task.UpdatedAt = time.Now()
	applyCancelReason(&current.Task, reason, "dead_letter_reason")
	return nil
}

func (q *MemoryQueue) ListDeadLetters(_ context.Context, limit int) ([]domain.Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	items := make([]*item, 0)
	for _, current := range q.items {
		if current.Task.State == domain.TaskDeadLetter {
			items = append(items, current)
		}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Task.UpdatedAt.Equal(items[j].Task.UpdatedAt) {
			return items[i].Task.ID < items[j].Task.ID
		}
		return items[i].Task.UpdatedAt.After(items[j].Task.UpdatedAt)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]domain.Task, 0, len(items))
	for _, current := range items {
		out = append(out, current.Task)
	}
	return out, nil
}

func (q *MemoryQueue) ReplayDeadLetter(_ context.Context, taskID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[taskID]
	if !ok {
		return ErrTaskNotFound
	}
	if current.Task.State != domain.TaskDeadLetter {
		return errors.New("task is not dead-lettered")
	}
	current.Leased = false
	current.LeaseWorker = ""
	current.LeaseExpires = time.Time{}
	current.AvailableAt = time.Now()
	current.Task.State = domain.TaskQueued
	current.Task.UpdatedAt = time.Now()
	return nil
}

func (q *MemoryQueue) GetTask(_ context.Context, taskID string) (TaskSnapshot, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	current, ok := q.items[taskID]
	if !ok {
		return TaskSnapshot{}, ErrTaskNotFound
	}
	return snapshotFromItem(current), nil
}

func (q *MemoryQueue) ListTasks(_ context.Context, limit int) ([]TaskSnapshot, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	q.recoverExpiredLeases(now)
	items := make([]*item, 0, len(q.items))
	for _, current := range q.items {
		items = append(items, current)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if rankForState(items[i].Task.State) == rankForState(items[j].Task.State) {
			if items[i].Task.Priority == items[j].Task.Priority {
				return items[i].Task.UpdatedAt.After(items[j].Task.UpdatedAt)
			}
			return items[i].Task.Priority < items[j].Task.Priority
		}
		return rankForState(items[i].Task.State) < rankForState(items[j].Task.State)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]TaskSnapshot, 0, len(items))
	for _, current := range items {
		out = append(out, snapshotFromItem(current))
	}
	return out, nil
}

func (q *MemoryQueue) CancelTask(_ context.Context, taskID string, reason string) (TaskSnapshot, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[taskID]
	if !ok {
		return TaskSnapshot{}, ErrTaskNotFound
	}
	if current.Task.State == domain.TaskDeadLetter {
		return TaskSnapshot{}, errors.New("task already dead-lettered")
	}
	now := time.Now()
	current.Task.State = domain.TaskCancelled
	current.Task.UpdatedAt = now
	applyCancelReason(&current.Task, reason, "cancel_reason")
	if !current.Leased {
		snapshot := snapshotFromItem(current)
		delete(q.items, taskID)
		return snapshot, nil
	}
	return snapshotFromItem(current), nil
}

func (q *MemoryQueue) Size(_ context.Context) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	count := 0
	for _, current := range q.items {
		if isActionableState(current.Task.State) {
			count++
		}
	}
	return count
}

func (q *MemoryQueue) recoverExpiredLeases(now time.Time) {
	for taskID, current := range q.items {
		if !current.Leased || current.LeaseExpires.After(now) {
			continue
		}
		if current.Task.State == domain.TaskCancelled {
			delete(q.items, taskID)
			continue
		}
		current.Leased = false
		current.LeaseWorker = ""
		current.Task.State = domain.TaskQueued
		current.Task.UpdatedAt = now
		current.AvailableAt = now
	}
}

func snapshotFromItem(current *item) TaskSnapshot {
	return TaskSnapshot{
		Task:         current.Task,
		AvailableAt:  current.AvailableAt,
		Attempt:      current.Attempt,
		Leased:       current.Leased,
		LeaseWorker:  current.LeaseWorker,
		LeaseExpires: current.LeaseExpires,
	}
}

func applyCancelReason(task *domain.Task, reason string, key string) {
	if reason == "" {
		return
	}
	if task.Metadata == nil {
		task.Metadata = make(map[string]string)
	}
	task.Metadata[key] = reason
}

func isActionableState(state domain.TaskState) bool {
	switch state {
	case domain.TaskCancelled, domain.TaskDeadLetter:
		return false
	default:
		return true
	}
}

func rankForState(state domain.TaskState) int {
	switch state {
	case domain.TaskQueued:
		return 0
	case domain.TaskLeased:
		return 1
	case domain.TaskRunning:
		return 2
	case domain.TaskRetrying:
		return 3
	case domain.TaskBlocked:
		return 4
	case domain.TaskCancelled:
		return 5
	case domain.TaskDeadLetter:
		return 6
	default:
		return 7
	}
}
