package queue

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"bigclaw-go/internal/domain"
)

type FileQueue struct {
	mu    sync.Mutex
	path  string
	items map[string]*item
}

func NewFileQueue(path string) (*FileQueue, error) {
	q := &FileQueue{path: path, items: make(map[string]*item)}
	if err := q.load(); err != nil {
		return nil, err
	}
	return q, nil
}

func (q *FileQueue) Enqueue(_ context.Context, task domain.Task) error {
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
		return q.save()
	}
	task.State = domain.TaskQueued
	task.UpdatedAt = now
	q.items[task.ID] = &item{Task: task, AvailableAt: now}
	return q.save()
}

func (q *FileQueue) LeaseNext(_ context.Context, workerID string, ttl time.Duration) (*domain.Task, *Lease, error) {
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
	picked.Task.UpdatedAt = now
	picked.Attempt++
	if err := q.save(); err != nil {
		return nil, nil, err
	}

	lease := &Lease{TaskID: picked.Task.ID, WorkerID: workerID, ExpiresAt: picked.LeaseExpires, Attempt: picked.Attempt, AcquiredAt: now}
	copy := picked.Task
	return &copy, lease, nil
}

func (q *FileQueue) RenewLease(_ context.Context, lease *Lease, ttl time.Duration) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[lease.TaskID]
	if !ok {
		return ErrTaskNotFound
	}
	if !current.Leased || current.LeaseWorker != lease.WorkerID {
		return errors.New("lease not owned by worker")
	}
	current.LeaseExpires = time.Now().Add(ttl)
	lease.ExpiresAt = current.LeaseExpires
	return q.save()
}

func (q *FileQueue) Ack(_ context.Context, lease *Lease) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if _, ok := q.items[lease.TaskID]; !ok {
		return ErrTaskNotFound
	}
	delete(q.items, lease.TaskID)
	return q.save()
}

func (q *FileQueue) Requeue(_ context.Context, lease *Lease, availableAt time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[lease.TaskID]
	if !ok {
		return ErrTaskNotFound
	}
	current.Leased = false
	current.LeaseWorker = ""
	current.LeaseExpires = time.Time{}
	current.AvailableAt = availableAt
	current.Task.State = domain.TaskQueued
	current.Task.UpdatedAt = time.Now()
	return q.save()
}

func (q *FileQueue) DeadLetter(_ context.Context, lease *Lease, reason string) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[lease.TaskID]
	if !ok {
		return ErrTaskNotFound
	}
	current.Leased = false
	current.LeaseWorker = ""
	current.LeaseExpires = time.Time{}
	current.AvailableAt = time.Now()
	current.Task.State = domain.TaskDeadLetter
	current.Task.UpdatedAt = time.Now()
	applyCancelReason(&current.Task, reason, "dead_letter_reason")
	return q.save()
}

func (q *FileQueue) ListDeadLetters(_ context.Context, limit int) ([]domain.Task, error) {
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

func (q *FileQueue) ReplayDeadLetter(_ context.Context, taskID string) error {
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
	return q.save()
}

func (q *FileQueue) GetTask(_ context.Context, taskID string) (TaskSnapshot, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	current, ok := q.items[taskID]
	if !ok {
		return TaskSnapshot{}, ErrTaskNotFound
	}
	return snapshotFromItem(current), nil
}

func (q *FileQueue) ListTasks(_ context.Context, limit int) ([]TaskSnapshot, error) {
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

func (q *FileQueue) CancelTask(_ context.Context, taskID string, reason string) (TaskSnapshot, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	current, ok := q.items[taskID]
	if !ok {
		return TaskSnapshot{}, ErrTaskNotFound
	}
	if current.Task.State == domain.TaskDeadLetter {
		return TaskSnapshot{}, errors.New("task already dead-lettered")
	}
	current.Task.State = domain.TaskCancelled
	current.Task.UpdatedAt = time.Now()
	applyCancelReason(&current.Task, reason, "cancel_reason")
	if !current.Leased {
		snapshot := snapshotFromItem(current)
		delete(q.items, taskID)
		return snapshot, q.save()
	}
	return snapshotFromItem(current), q.save()
}

func (q *FileQueue) Size(_ context.Context) int {
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

func (q *FileQueue) load() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(q.path), 0o755); err != nil {
		return err
	}
	contents, err := os.ReadFile(q.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if len(contents) == 0 {
		return nil
	}
	return json.Unmarshal(contents, &q.items)
}

func (q *FileQueue) save() error {
	contents, err := json.MarshalIndent(q.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := q.path + ".tmp"
	if err := os.WriteFile(tmp, contents, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, q.path)
}

func (q *FileQueue) recoverExpiredLeases(now time.Time) {
	for _, current := range q.items {
		if current.Leased && !current.LeaseExpires.After(now) && current.Task.State != domain.TaskCancelled {
			current.Leased = false
			current.LeaseWorker = ""
			current.Task.State = domain.TaskQueued
			current.Task.UpdatedAt = now
			current.AvailableAt = now
		}
	}
}
