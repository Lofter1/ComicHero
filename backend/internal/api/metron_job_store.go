package api

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type metronImportJobStore struct {
	nextID           atomic.Uint64
	nextSubscriberID atomic.Uint64
	mu               sync.RWMutex
	jobs             map[string]*MetronImportJob
	cancel           map[string]context.CancelFunc
	subscribers      map[uint64]chan MetronImportJob
	queue            chan queuedMetronImportJob
}

type queuedMetronImportJob struct {
	id      string
	ctx     context.Context
	message string
	run     func(context.Context, func(int, int, string)) error
}

func newMetronImportJobStore() *metronImportJobStore {
	store := &metronImportJobStore{
		jobs:        map[string]*MetronImportJob{},
		cancel:      map[string]context.CancelFunc{},
		subscribers: map[uint64]chan MetronImportJob{},
		queue:       make(chan queuedMetronImportJob, 100),
	}
	go store.runQueue()
	return store
}

func NewMetronImportJobStore() *metronImportJobStore {
	return newMetronImportJobStore()
}

func (s *metronImportJobStore) start(jobType string, metronID int, message string, run func(context.Context, func(int, int, string)) error) MetronImportJob {
	return s.startWithOptions(jobType, metronID, defaultMetronImportOptions(), message, run)
}

func (s *metronImportJobStore) startWithOptions(jobType string, metronID int, options MetronImportOptions, message string, run func(context.Context, func(int, int, string)) error) MetronImportJob {
	return s.startWithContextAndOptions(context.Background(), jobType, metronID, options, message, run)
}

func (s *metronImportJobStore) startWithContextAndOptions(parent context.Context, jobType string, metronID int, options MetronImportOptions, message string, run func(context.Context, func(int, int, string)) error) MetronImportJob {
	id := fmt.Sprintf("metron-%d", s.nextID.Add(1))
	ctx, cancel := context.WithCancel(context.WithoutCancel(parent))
	job := &MetronImportJob{
		ID:        id,
		Type:      jobType,
		MetronID:  metronID,
		Options:   options,
		Status:    "queued",
		Message:   message,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	s.mu.Lock()
	s.jobs[id] = job
	s.cancel[id] = cancel
	s.mu.Unlock()
	s.broadcast(*job)

	s.queue <- queuedMetronImportJob{id: id, ctx: ctx, message: message, run: run}

	return *job
}

func (s *metronImportJobStore) runQueue() {
	for queued := range s.queue {
		s.runQueued(queued)
	}
}

func (s *metronImportJobStore) runQueued(queued queuedMetronImportJob) {
	defer s.clearCancel(queued.id)

	if err := queued.ctx.Err(); err != nil {
		s.update(queued.id, "canceled", "Import canceled before it started.", time.Now().UTC().Format(time.RFC3339), -1, -1)
		return
	}

	s.update(queued.id, "running", queued.message, "", 0, 0)
	if err := queued.run(queued.ctx, func(completed, total int, message string) {
		s.updateProgress(queued.id, completed, total, message)
	}); err != nil {
		if isContextCanceledError(err) {
			s.update(queued.id, "canceled", "Import canceled.", time.Now().UTC().Format(time.RFC3339), -1, -1)
			return
		}
		job, _ := s.get(queued.id)
		log.Printf("metron import job %s failed: type=%s metron_id=%d mode=%s force=%t error=%v", queued.id, job.Type, job.MetronID, job.Options.Mode, job.Options.Force, err)
		s.update(queued.id, "failed", err.Error(), time.Now().UTC().Format(time.RFC3339), -1, -1)
		return
	}
	if err := queued.ctx.Err(); err != nil {
		s.update(queued.id, "canceled", "Import canceled.", time.Now().UTC().Format(time.RFC3339), -1, -1)
		return
	}

	job, ok := s.get(queued.id)
	if ok {
		s.update(queued.id, "succeeded", successMessage(job.Type), time.Now().UTC().Format(time.RFC3339), -1, -1)
	}
}

func (s *metronImportJobStore) list() []MetronImportJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]MetronImportJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, *job)
	}
	sortMetronImportJobs(jobs)
	return jobs
}

func (s *metronImportJobStore) get(id string) (MetronImportJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return MetronImportJob{}, false
	}
	return *job, true
}

func (s *metronImportJobStore) cancelJob(id string) (MetronImportJob, bool) {
	s.mu.Lock()

	job, ok := s.jobs[id]
	if !ok {
		s.mu.Unlock()
		return MetronImportJob{}, false
	}
	if job.Status != "queued" && job.Status != "running" {
		next := *job
		s.mu.Unlock()
		return next, true
	}
	if cancel, ok := s.cancel[id]; ok {
		cancel()
	}
	if job.Status == "queued" {
		job.Status = "canceled"
		job.Message = "Import canceled before it started."
		job.EndedAt = time.Now().UTC().Format(time.RFC3339)
		next := *job
		s.mu.Unlock()
		s.broadcast(next)
		return next, true
	}
	job.Status = "canceling"
	job.Message = "Canceling import..."
	next := *job
	s.mu.Unlock()
	s.broadcast(next)
	return next, true
}

func (s *metronImportJobStore) clearCancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cancel, id)
}

func (s *metronImportJobStore) update(id, status, message, endedAt string, completed, total int) {
	s.mu.Lock()

	if job, ok := s.jobs[id]; ok {
		job.Status = status
		job.Message = message
		job.EndedAt = endedAt
		if completed >= 0 {
			job.Completed = completed
		}
		if total >= 0 {
			job.Total = total
		}
		next := *job
		s.mu.Unlock()
		s.broadcast(next)
		return
	}
	s.mu.Unlock()
}

func (s *metronImportJobStore) updateProgress(id string, completed, total int, message string) {
	s.mu.Lock()

	if job, ok := s.jobs[id]; ok {
		job.Completed = completed
		job.Total = total
		if message != "" {
			job.Message = message
		}
		next := *job
		s.mu.Unlock()
		s.broadcast(next)
		return
	}
	s.mu.Unlock()
}

func (s *metronImportJobStore) deleteTerminal(id string) (MetronImportJob, bool, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return MetronImportJob{}, false, false
	}
	if job.Status == "queued" || job.Status == "running" || job.Status == "canceling" {
		return *job, true, false
	}
	delete(s.jobs, id)
	return *job, true, true
}

func (s *metronImportJobStore) subscribe() (<-chan MetronImportJob, func()) {
	id := s.nextSubscriberID.Add(1)

	s.mu.Lock()
	ch := make(chan MetronImportJob, len(s.jobs)+32)
	s.subscribers[id] = ch
	jobs := make([]MetronImportJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, *job)
	}
	s.mu.Unlock()

	sortMetronImportJobs(jobs)
	for _, job := range jobs {
		ch <- job
	}

	unsubscribe := func() {
		s.mu.Lock()
		if current, ok := s.subscribers[id]; ok {
			delete(s.subscribers, id)
			close(current)
		}
		s.mu.Unlock()
	}
	return ch, unsubscribe
}

func (s *metronImportJobStore) broadcast(job MetronImportJob) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- job:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- job:
			default:
			}
		}
	}
}

func sortMetronImportJobs(jobs []MetronImportJob) {
	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].StartedAt > jobs[j].StartedAt
	})
}

func successMessage(jobType string) string {
	switch jobType {
	case "comic":
		return "Comic import finished."
	case "readingList":
		return "Reading list import finished."
	case "readingLists":
		return "All reading lists imported."
	case "arc":
		return "Arc import finished."
	case "series":
		return "Series import finished."
	case "character":
		return "Character import finished."
	default:
		return "Import finished."
	}
}
