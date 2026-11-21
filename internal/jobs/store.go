package jobs

import (
	"context"
	"sync"
	"time"

	"atad-project/internal/gitlog"
	"atad-project/internal/stats"
)

type Job struct {
	ID       int             `json:"id"`
	State    string          `json:"state"`
	Policy   string          `json:"merge_policy"`
	Commits  int             `json:"commits"`
	Duration int64           `json:"analysed_in_ms"`
	Message  string          `json:"message,omitempty"`
	History  []gitlog.Commit `json:"-"`
}

type entry struct {
	job    Job
	cancel context.CancelFunc
}

type Store struct {
	mu               sync.RWMutex
	jobs             map[int]*entry
	next             int
	initial, aliases stats.Aliases
}

func New(aliases stats.Aliases) *Store {
	return &Store{jobs: make(map[int]*entry), next: 1, initial: aliases, aliases: aliases}
}

// Start returns the assigned ID immediately and publishes the history only after the read succeeds.
func (s *Store) Start(policy string, read func(context.Context, func(gitlog.Commit)) error) Job {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	job := Job{ID: s.next, State: "analysing", Policy: policy}
	e := &entry{job: job, cancel: cancel}
	s.jobs[job.ID] = e
	s.next++
	s.mu.Unlock()
	go func() {
		defer cancel()
		start := time.Now()
		var history []gitlog.Commit
		err := read(ctx, func(c gitlog.Commit) {
			if policy == "count" || !c.Merge {
				history = append(history, c)
			}
		})
		s.mu.Lock()
		defer s.mu.Unlock()
		if ctx.Err() != nil {
			return
		}
		e.job.Duration = time.Since(start).Milliseconds()
		if err != nil {
			e.job.State = "failed"
			e.job.Message = err.Error()
			return
		}
		e.job.History = history
		e.job.Commits = len(history)
		e.job.State = "ready"
	}()
	return job
}

func (s *Store) Get(id int) (Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.jobs[id]
	if !ok {
		return Job{}, false
	}
	return e.job, true
}

func (s *Store) Aliases() stats.Aliases {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.aliases
}

func (s *Store) SetAliases(a stats.Aliases) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.aliases = a
}

// Reset cancels active reads, releases stored snapshots and starts ID allocation over.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.jobs {
		e.cancel()
	}
	clear(s.jobs)
	s.next = 1
	s.aliases = s.initial
}
