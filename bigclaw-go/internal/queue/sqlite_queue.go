package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"bigclaw-go/internal/domain"
	_ "modernc.org/sqlite"
)

type SQLiteQueue struct {
	db   *sql.DB
	path string
}

func NewSQLiteQueue(path string) (*SQLiteQueue, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	q := &SQLiteQueue{db: db, path: path}
	if err := q.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return q, nil
}

func (q *SQLiteQueue) init() error {
	stmts := []string{
		`PRAGMA journal_mode=WAL;`,
		`PRAGMA busy_timeout=5000;`,
		`CREATE TABLE IF NOT EXISTS tasks (
			task_id TEXT PRIMARY KEY,
			payload BLOB NOT NULL,
			priority INTEGER NOT NULL,
			created_at_ns INTEGER NOT NULL,
			state TEXT NOT NULL,
			available_at_ns INTEGER NOT NULL,
			attempt INTEGER NOT NULL,
			leased INTEGER NOT NULL,
			lease_worker TEXT NOT NULL,
			lease_expires_ns INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_available ON tasks(leased, available_at_ns, priority, created_at_ns);`,
	}
	for _, stmt := range stmts {
		if _, err := q.db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (q *SQLiteQueue) Enqueue(_ context.Context, task domain.Task) error {
	now := time.Now()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	task.State = domain.TaskQueued
	payload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	_, err = q.db.Exec(`INSERT INTO tasks(task_id, payload, priority, created_at_ns, state, available_at_ns, attempt, leased, lease_worker, lease_expires_ns)
		VALUES(?, ?, ?, ?, ?, ?, 0, 0, '', 0)
		ON CONFLICT(task_id) DO UPDATE SET payload=excluded.payload, priority=excluded.priority, created_at_ns=excluded.created_at_ns, state=excluded.state, available_at_ns=excluded.available_at_ns, leased=0, lease_worker='', lease_expires_ns=0`,
		task.ID, payload, task.Priority, task.CreatedAt.UnixNano(), string(task.State), now.UnixNano())
	return err
}

func (q *SQLiteQueue) LeaseNext(_ context.Context, workerID string, ttl time.Duration) (*domain.Task, *Lease, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now()
	row := tx.QueryRow(`SELECT task_id, payload, attempt FROM tasks
		WHERE available_at_ns <= ?
		AND state NOT IN (?, ?)
		AND (leased = 0 OR lease_expires_ns <= ?)
		ORDER BY priority ASC, created_at_ns ASC
		LIMIT 1`, now.UnixNano(), string(domain.TaskDeadLetter), string(domain.TaskCancelled), now.UnixNano())

	var taskID string
	var payload []byte
	var attempt int
	if err := row.Scan(&taskID, &payload, &attempt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			_ = tx.Rollback()
			tx = nil
			return nil, nil, nil
		}
		return nil, nil, err
	}

	var task domain.Task
	if err := json.Unmarshal(payload, &task); err != nil {
		return nil, nil, err
	}
	task.State = domain.TaskLeased
	task.UpdatedAt = now
	attempt++
	leaseExpires := now.Add(ttl)
	updatedPayload, err := json.Marshal(task)
	if err != nil {
		return nil, nil, err
	}
	result, err := tx.Exec(`UPDATE tasks SET payload=?, state=?, leased=1, lease_worker=?, lease_expires_ns=?, attempt=? WHERE task_id=?`, updatedPayload, string(task.State), workerID, leaseExpires.UnixNano(), attempt, taskID)
	if err != nil {
		return nil, nil, err
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		return nil, nil, fmt.Errorf("unexpected rows affected during lease: %d", rows)
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	tx = nil
	return &task, &Lease{TaskID: taskID, WorkerID: workerID, ExpiresAt: leaseExpires, Attempt: attempt, AcquiredAt: now}, nil
}

func (q *SQLiteQueue) RenewLease(_ context.Context, lease *Lease, ttl time.Duration) error {
	now := time.Now()
	expiresAt := now.Add(ttl)
	result, err := q.db.Exec(`UPDATE tasks SET lease_expires_ns=? WHERE task_id=? AND leased=1 AND lease_worker=? AND attempt=? AND lease_expires_ns > ?`, expiresAt.UnixNano(), lease.TaskID, lease.WorkerID, lease.Attempt, now.UnixNano())
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		row := q.db.QueryRow(`SELECT lease_expires_ns FROM tasks WHERE task_id=? AND leased=1 AND lease_worker=? AND attempt=?`, lease.TaskID, lease.WorkerID, lease.Attempt)
		var leaseExpiresNS int64
		if err := row.Scan(&leaseExpiresNS); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrLeaseNotOwned
			}
			return err
		}
		if leaseExpiresNS <= now.UnixNano() {
			return ErrLeaseExpired
		}
		return ErrLeaseNotOwned
	}
	lease.ExpiresAt = expiresAt
	return nil
}

func (q *SQLiteQueue) Ack(_ context.Context, lease *Lease) error {
	result, err := q.db.Exec(`DELETE FROM tasks WHERE task_id=? AND leased=1 AND lease_worker=? AND attempt=?`, lease.TaskID, lease.WorkerID, lease.Attempt)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		if _, err := q.GetTask(context.Background(), lease.TaskID); err != nil {
			if errors.Is(err, ErrTaskNotFound) {
				return ErrTaskNotFound
			}
			return err
		}
		return ErrLeaseNotOwned
	}
	return nil
}

func (q *SQLiteQueue) Requeue(_ context.Context, lease *Lease, availableAt time.Time) error {
	return q.updateStateAfterLease(lease, domain.TaskQueued, availableAt, "")
}

func (q *SQLiteQueue) DeadLetter(_ context.Context, lease *Lease, reason string) error {
	return q.updateStateAfterLease(lease, domain.TaskDeadLetter, time.Now(), reason)
}

func (q *SQLiteQueue) ListDeadLetters(_ context.Context, limit int) ([]domain.Task, error) {
	query := `SELECT payload FROM tasks WHERE state = ? ORDER BY available_at_ns DESC, created_at_ns DESC`
	args := []any{string(domain.TaskDeadLetter)}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Task, 0)
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var task domain.Task
		if err := json.Unmarshal(payload, &task); err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

func (q *SQLiteQueue) ReplayDeadLetter(_ context.Context, taskID string) error {
	row := q.db.QueryRow(`SELECT payload, attempt FROM tasks WHERE task_id=?`, taskID)
	var payload []byte
	var attempt int
	if err := row.Scan(&payload, &attempt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return err
	}
	var task domain.Task
	if err := json.Unmarshal(payload, &task); err != nil {
		return err
	}
	if task.State != domain.TaskDeadLetter {
		return errors.New("task is not dead-lettered")
	}
	now := time.Now()
	task.State = domain.TaskQueued
	task.UpdatedAt = now
	updatedPayload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	_, err = q.db.Exec(`UPDATE tasks SET payload=?, state=?, available_at_ns=?, leased=0, lease_worker='', lease_expires_ns=0, attempt=? WHERE task_id=?`, updatedPayload, string(domain.TaskQueued), now.UnixNano(), attempt, taskID)
	return err
}

func (q *SQLiteQueue) GetTask(_ context.Context, taskID string) (TaskSnapshot, error) {
	row := q.db.QueryRow(`SELECT payload, available_at_ns, attempt, leased, lease_worker, lease_expires_ns FROM tasks WHERE task_id=?`, taskID)
	return scanSnapshotRow(row)
}

func (q *SQLiteQueue) ListTasks(_ context.Context, limit int) ([]TaskSnapshot, error) {
	rows, err := q.db.Query(`SELECT payload, available_at_ns, attempt, leased, lease_worker, lease_expires_ns FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]TaskSnapshot, 0)
	for rows.Next() {
		snapshot, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		if rankForState(out[i].Task.State) == rankForState(out[j].Task.State) {
			if out[i].Task.Priority == out[j].Task.Priority {
				return out[i].Task.UpdatedAt.After(out[j].Task.UpdatedAt)
			}
			return out[i].Task.Priority < out[j].Task.Priority
		}
		return rankForState(out[i].Task.State) < rankForState(out[j].Task.State)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (q *SQLiteQueue) CancelTask(_ context.Context, taskID string, reason string) (TaskSnapshot, error) {
	row := q.db.QueryRow(`SELECT payload, available_at_ns, attempt, leased, lease_worker, lease_expires_ns FROM tasks WHERE task_id=?`, taskID)
	snapshot, err := scanSnapshotRow(row)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if snapshot.Task.State == domain.TaskDeadLetter {
		return TaskSnapshot{}, errors.New("task already dead-lettered")
	}
	snapshot.Task.State = domain.TaskCancelled
	snapshot.Task.UpdatedAt = time.Now()
	applyCancelReason(&snapshot.Task, reason, "cancel_reason")
	payload, err := json.Marshal(snapshot.Task)
	if err != nil {
		return TaskSnapshot{}, err
	}
	if !snapshot.Leased {
		if _, err := q.db.Exec(`DELETE FROM tasks WHERE task_id=?`, taskID); err != nil {
			return TaskSnapshot{}, err
		}
		return snapshot, nil
	}
	_, err = q.db.Exec(`UPDATE tasks SET payload=?, state=? WHERE task_id=?`, payload, string(domain.TaskCancelled), taskID)
	if err != nil {
		return TaskSnapshot{}, err
	}
	return snapshot, nil
}

func (q *SQLiteQueue) updateStateAfterLease(lease *Lease, state domain.TaskState, availableAt time.Time, reason string) error {
	row := q.db.QueryRow(`SELECT payload FROM tasks WHERE task_id=?`, lease.TaskID)
	var payload []byte
	if err := row.Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrTaskNotFound
		}
		return err
	}
	var task domain.Task
	if err := json.Unmarshal(payload, &task); err != nil {
		return err
	}
	task.State = state
	task.UpdatedAt = time.Now()
	if state == domain.TaskDeadLetter {
		applyCancelReason(&task, reason, "dead_letter_reason")
	}
	updatedPayload, err := json.Marshal(task)
	if err != nil {
		return err
	}
	result, err := q.db.Exec(`UPDATE tasks SET payload=?, state=?, available_at_ns=?, leased=0, lease_worker='', lease_expires_ns=0 WHERE task_id=? AND leased=1 AND lease_worker=? AND attempt=?`, updatedPayload, string(state), availableAt.UnixNano(), lease.TaskID, lease.WorkerID, lease.Attempt)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows != 1 {
		if _, err := q.GetTask(context.Background(), lease.TaskID); err != nil {
			if errors.Is(err, ErrTaskNotFound) {
				return ErrTaskNotFound
			}
			return err
		}
		return ErrLeaseNotOwned
	}
	return nil
}

func (q *SQLiteQueue) Size(_ context.Context) int {
	row := q.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE state NOT IN (?, ?)`, string(domain.TaskDeadLetter), string(domain.TaskCancelled))
	var count int
	if err := row.Scan(&count); err != nil {
		return 0
	}
	return count
}

func scanSnapshotRow(row *sql.Row) (TaskSnapshot, error) {
	var payload []byte
	var availableAtNS int64
	var attempt int
	var leased int
	var leaseWorker string
	var leaseExpiresNS int64
	if err := row.Scan(&payload, &availableAtNS, &attempt, &leased, &leaseWorker, &leaseExpiresNS); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskSnapshot{}, ErrTaskNotFound
		}
		return TaskSnapshot{}, err
	}
	return buildSnapshot(payload, availableAtNS, attempt, leased, leaseWorker, leaseExpiresNS)
}

func scanSnapshot(rows *sql.Rows) (TaskSnapshot, error) {
	var payload []byte
	var availableAtNS int64
	var attempt int
	var leased int
	var leaseWorker string
	var leaseExpiresNS int64
	if err := rows.Scan(&payload, &availableAtNS, &attempt, &leased, &leaseWorker, &leaseExpiresNS); err != nil {
		return TaskSnapshot{}, err
	}
	return buildSnapshot(payload, availableAtNS, attempt, leased, leaseWorker, leaseExpiresNS)
}

func buildSnapshot(payload []byte, availableAtNS int64, attempt int, leased int, leaseWorker string, leaseExpiresNS int64) (TaskSnapshot, error) {
	var task domain.Task
	if err := json.Unmarshal(payload, &task); err != nil {
		return TaskSnapshot{}, err
	}
	return TaskSnapshot{
		Task:         task,
		AvailableAt:  time.Unix(0, availableAtNS),
		Attempt:      attempt,
		Leased:       leased == 1,
		LeaseWorker:  leaseWorker,
		LeaseExpires: time.Unix(0, leaseExpiresNS),
	}, nil
}

func (q *SQLiteQueue) Close() error { return q.db.Close() }
