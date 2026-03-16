package control

import (
	"sort"
	"sync"
	"time"
)

type Note struct {
	Actor     string    `json:"actor,omitempty"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

type Takeover struct {
	TaskID     string    `json:"task_id"`
	Active     bool      `json:"active"`
	Owner      string    `json:"owner,omitempty"`
	Reviewer   string    `json:"reviewer,omitempty"`
	StartedAt  time.Time `json:"started_at,omitempty"`
	ReleasedAt time.Time `json:"released_at,omitempty"`
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
	Notes      []Note    `json:"notes,omitempty"`
}

type Snapshot struct {
	Paused          bool      `json:"paused"`
	PauseReason     string    `json:"pause_reason,omitempty"`
	PauseActor      string    `json:"pause_actor,omitempty"`
	PausedAt        time.Time `json:"paused_at,omitempty"`
	ActiveTakeovers int       `json:"active_takeovers"`
}

type Controller struct {
	mu          sync.Mutex
	paused      bool
	pauseReason string
	pauseActor  string
	pausedAt    time.Time
	takeovers   map[string]Takeover
}

func New() *Controller {
	return &Controller{takeovers: make(map[string]Takeover)}
}

func (c *Controller) Pause(actor, reason string, at time.Time) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at.IsZero() {
		at = time.Now()
	}
	c.paused = true
	c.pauseReason = reason
	c.pauseActor = actor
	c.pausedAt = at
	return c.snapshotLocked()
}

func (c *Controller) Resume(_ string, at time.Time) Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at.IsZero() {
		at = time.Now()
	}
	c.paused = false
	c.pauseReason = ""
	c.pauseActor = ""
	c.pausedAt = time.Time{}
	return c.snapshotLocked()
}

func (c *Controller) IsPaused() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

func (c *Controller) Snapshot() Snapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.snapshotLocked()
}

func (c *Controller) Takeover(taskID, actor, reviewer, note string, at time.Time) Takeover {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at.IsZero() {
		at = time.Now()
	}
	takeover := c.takeovers[taskID]
	takeover.TaskID = taskID
	takeover.Active = true
	if takeover.StartedAt.IsZero() {
		takeover.StartedAt = at
	}
	takeover.ReleasedAt = time.Time{}
	takeover.UpdatedAt = at
	if actor != "" {
		takeover.Owner = actor
	}
	if reviewer != "" {
		takeover.Reviewer = reviewer
	}
	if note != "" {
		takeover.Notes = append(takeover.Notes, Note{Actor: actor, Message: note, Timestamp: at})
	}
	c.takeovers[taskID] = takeover
	return cloneTakeover(takeover)
}

func (c *Controller) Release(taskID, actor, note string, at time.Time) (Takeover, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at.IsZero() {
		at = time.Now()
	}
	takeover, ok := c.takeovers[taskID]
	if !ok {
		return Takeover{}, false
	}
	takeover.Active = false
	takeover.UpdatedAt = at
	takeover.ReleasedAt = at
	if takeover.Owner == "" && actor != "" {
		takeover.Owner = actor
	}
	if note != "" {
		takeover.Notes = append(takeover.Notes, Note{Actor: actor, Message: note, Timestamp: at})
	}
	c.takeovers[taskID] = takeover
	return cloneTakeover(takeover), true
}

func (c *Controller) Annotate(taskID, actor, note string, at time.Time) Takeover {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at.IsZero() {
		at = time.Now()
	}
	takeover := c.takeovers[taskID]
	takeover.TaskID = taskID
	takeover.UpdatedAt = at
	if takeover.Owner == "" && actor != "" {
		takeover.Owner = actor
	}
	if note != "" {
		takeover.Notes = append(takeover.Notes, Note{Actor: actor, Message: note, Timestamp: at})
	}
	c.takeovers[taskID] = takeover
	return cloneTakeover(takeover)
}

func (c *Controller) Reassign(taskID, owner, reviewer, actor, note string, at time.Time) (Takeover, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if at.IsZero() {
		at = time.Now()
	}
	takeover, ok := c.takeovers[taskID]
	if !ok || !takeover.Active {
		return Takeover{}, false
	}
	takeover.UpdatedAt = at
	if owner != "" {
		takeover.Owner = owner
	}
	if reviewer != "" {
		takeover.Reviewer = reviewer
	}
	if note != "" {
		takeover.Notes = append(takeover.Notes, Note{Actor: actor, Message: note, Timestamp: at})
	}
	c.takeovers[taskID] = takeover
	return cloneTakeover(takeover), true
}

func (c *Controller) TakeoverStatus(taskID string) (Takeover, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	takeover, ok := c.takeovers[taskID]
	if !ok {
		return Takeover{}, false
	}
	return cloneTakeover(takeover), true
}

func (c *Controller) ActiveTakeovers() []Takeover {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Takeover, 0)
	for _, takeover := range c.takeovers {
		if takeover.Active {
			out = append(out, cloneTakeover(takeover))
		}
	}
	sortTakeovers(out)
	return out
}

func (c *Controller) TakeoverHistory() []Takeover {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Takeover, 0, len(c.takeovers))
	for _, takeover := range c.takeovers {
		if !takeover.Active && takeover.StartedAt.IsZero() && takeover.ReleasedAt.IsZero() {
			continue
		}
		out = append(out, cloneTakeover(takeover))
	}
	sortTakeovers(out)
	return out
}

func (c *Controller) snapshotLocked() Snapshot {
	active := 0
	for _, takeover := range c.takeovers {
		if takeover.Active {
			active++
		}
	}
	return Snapshot{
		Paused:          c.paused,
		PauseReason:     c.pauseReason,
		PauseActor:      c.pauseActor,
		PausedAt:        c.pausedAt,
		ActiveTakeovers: active,
	}
}

func cloneTakeover(takeover Takeover) Takeover {
	clone := takeover
	if len(takeover.Notes) > 0 {
		clone.Notes = append([]Note(nil), takeover.Notes...)
	}
	return clone
}

func sortTakeovers(takeovers []Takeover) {
	sort.SliceStable(takeovers, func(i, j int) bool {
		if takeovers[i].UpdatedAt.Equal(takeovers[j].UpdatedAt) {
			return takeovers[i].TaskID < takeovers[j].TaskID
		}
		return takeovers[i].UpdatedAt.After(takeovers[j].UpdatedAt)
	})
}
