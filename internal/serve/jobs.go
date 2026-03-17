package serve

import (
	"sync"
	"time"

	"github.com/gongoeloe/grapher/internal/analyzer"
	"github.com/gongoeloe/grapher/internal/graph"
)

type JobStatus string

const (
	JobPending JobStatus = "pending"
	JobRunning JobStatus = "running"
	JobDone    JobStatus = "done"
	JobError   JobStatus = "error"
)

// Job represents an analysis job.
type Job struct {
	ID          string
	Status      JobStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
	Error       string
	Findings    []analyzer.Finding
	Graph       *graph.DiGraph
}

// JobStore is a thread-safe in-memory job store.
type JobStore struct {
	mu      sync.RWMutex
	jobs    map[string]*Job
	current string // ID of the currently running or most recent job
}

func NewJobStore() *JobStore {
	return &JobStore{jobs: make(map[string]*Job)}
}

func (s *JobStore) Add(j *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[j.ID] = j
}

func (s *JobStore) Get(id string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[id]
	return j, ok
}

func (s *JobStore) Update(id string, fn func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[id]; ok {
		fn(j)
	}
}

func (s *JobStore) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current == "" {
		return false
	}
	j, ok := s.jobs[s.current]
	return ok && j.Status == JobRunning
}

func (s *JobStore) SetCurrent(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current = id
}

func (s *JobStore) LatestDone() *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.current == "" {
		return nil
	}
	j, ok := s.jobs[s.current]
	if !ok || j.Status != JobDone {
		return nil
	}
	return j
}
