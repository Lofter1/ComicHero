package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

type MetronImportJob struct {
	ID        string `json:"id" doc:"Import job identifier." example:"metron-1"`
	Type      string `json:"type" doc:"Import type." enum:"comic,readingList,series" example:"series"`
	MetronID  int    `json:"metronId" doc:"Metron resource identifier." example:"123456"`
	Status    string `json:"status" doc:"Current job status." enum:"queued,running,succeeded,failed,canceled" example:"running"`
	Message   string `json:"message" doc:"Human-readable status message." example:"Importing series from Metron..."`
	Completed int    `json:"completed" doc:"Completed import units." example:"12"`
	Total     int    `json:"total" doc:"Total import units, when known." example:"52"`
	StartedAt string `json:"startedAt" doc:"RFC3339 start timestamp." example:"2026-06-26T12:30:00Z"`
	EndedAt   string `json:"endedAt,omitempty" doc:"RFC3339 end timestamp, when finished." example:"2026-06-26T12:31:00Z"`
}

type MetronImportJobOutput struct {
	Body MetronImportJob
}

type MetronImportJobInput struct {
	ID string `path:"id" doc:"Import job identifier." example:"metron-1"`
}

type metronImportJobStore struct {
	nextID atomic.Uint64
	mu     sync.RWMutex
	jobs   map[string]*MetronImportJob
	cancel map[string]context.CancelFunc
}

func newMetronImportJobStore() *metronImportJobStore {
	return &metronImportJobStore{
		jobs:   map[string]*MetronImportJob{},
		cancel: map[string]context.CancelFunc{},
	}
}

func NewMetronImportJobStore() *metronImportJobStore {
	return newMetronImportJobStore()
}

func (s *metronImportJobStore) start(jobType string, metronID int, message string, run func(context.Context, func(int, int, string)) error) MetronImportJob {
	id := fmt.Sprintf("metron-%d", s.nextID.Add(1))
	ctx, cancel := context.WithCancel(context.Background())
	job := &MetronImportJob{
		ID:        id,
		Type:      jobType,
		MetronID:  metronID,
		Status:    "queued",
		Message:   message,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	}

	s.mu.Lock()
	s.jobs[id] = job
	s.cancel[id] = cancel
	s.mu.Unlock()

	go func() {
		defer s.clearCancel(id)

		s.update(id, "running", message, "", 0, 0)
		if err := run(ctx, func(completed, total int, message string) {
			s.updateProgress(id, completed, total, message)
		}); err != nil {
			if isContextCanceledError(err) {
				s.update(id, "canceled", "Import canceled.", time.Now().UTC().Format(time.RFC3339), -1, -1)
				return
			}
			s.update(id, "failed", err.Error(), time.Now().UTC().Format(time.RFC3339), -1, -1)
			return
		}
		s.update(id, "succeeded", successMessage(jobType), time.Now().UTC().Format(time.RFC3339), -1, -1)
	}()

	return *job
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
	defer s.mu.Unlock()

	job, ok := s.jobs[id]
	if !ok {
		return MetronImportJob{}, false
	}
	if job.Status != "queued" && job.Status != "running" {
		return *job, true
	}
	if cancel, ok := s.cancel[id]; ok {
		cancel()
	}
	job.Message = "Canceling import..."
	return *job, true
}

func (s *metronImportJobStore) clearCancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.cancel, id)
}

func (s *metronImportJobStore) update(id, status, message, endedAt string, completed, total int) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	}
}

func (s *metronImportJobStore) updateProgress(id string, completed, total int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job, ok := s.jobs[id]; ok {
		job.Completed = completed
		job.Total = total
		if message != "" {
			job.Message = message
		}
	}
}

func successMessage(jobType string) string {
	switch jobType {
	case "comic":
		return "Comic import finished."
	case "readingList":
		return "Reading list import finished."
	case "series":
		return "Series import finished."
	default:
		return "Import finished."
	}
}

func startMetronComicImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return store.start("comic", metronID, "Importing comic from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 1, "Checking existing imports...")
		if _, ok, err := existingComicIDByMetronIssueID(ctx, db, metronID); err != nil || ok {
			if ok {
				progress(1, 1, "Comic already exists.")
			}
			return err
		}

		progress(0, 1, "Fetching comic from Metron...")
		issue, err := client.GetIssue(ctx, metronID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		_, err = importMetronComic(ctx, db, covers, *issue)
		if err == nil {
			progress(1, 1, "Comic imported.")
		}
		return err
	})
}

func startMetronReadingListImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return store.start("readingList", metronID, "Importing reading list and issues from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Checking existing imports...")
		if _, ok, err := existingReadingOrderIDByMetronID(ctx, db, metronID); err != nil || ok {
			if ok {
				progress(1, 1, "Reading list already exists.")
			}
			return err
		}

		progress(0, 0, "Fetching reading list from Metron...")
		list, err := client.GetReadingList(ctx, metronID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		return importMetronReadingListWithProgress(ctx, db, covers, *list, progress)
	})
}

func startMetronSeriesImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return store.start("series", metronID, "Importing series from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching series issue list from Metron...")
		issues, err := client.GetSeriesIssues(ctx, metronID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		_, err = importMetronSeriesWithProgress(ctx, db, client, covers, issues, progress)
		return err
	})
}

func metronImportError(err error) error {
	if isContextCanceledError(err) {
		return err
	}
	return metronAPIError(err)
}

func isContextCanceledError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || strings.Contains(strings.ToLower(err.Error()), "context canceled")
}

func getMetronImportJob(store *metronImportJobStore, id string) (*MetronImportJobOutput, error) {
	job, ok := store.get(id)
	if !ok {
		return nil, huma.Error404NotFound("import job not found")
	}
	return &MetronImportJobOutput{Body: job}, nil
}

func cancelMetronImportJob(store *metronImportJobStore, id string) (*MetronImportJobOutput, error) {
	job, ok := store.cancelJob(id)
	if !ok {
		return nil, huma.Error404NotFound("import job not found")
	}
	return &MetronImportJobOutput{Body: job}, nil
}

func continueMetronImportJob(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, id string) (*MetronImportJobOutput, error) {
	job, ok := store.get(id)
	if !ok {
		return nil, huma.Error404NotFound("import job not found")
	}
	if job.Status != "canceled" {
		return nil, huma.Error400BadRequest("only canceled imports can be continued")
	}

	var next MetronImportJob
	switch job.Type {
	case "comic":
		next = startMetronComicImport(store, db, client, covers, job.MetronID)
	case "readingList":
		next = startMetronReadingListContinue(store, db, client, covers, job.MetronID)
	case "series":
		next = startMetronSeriesImport(store, db, client, covers, job.MetronID)
	default:
		return nil, huma.Error400BadRequest("unsupported import type")
	}
	return &MetronImportJobOutput{Body: next}, nil
}

func startMetronReadingListContinue(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return store.start("readingList", metronID, "Continuing reading list import from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching reading list from Metron...")
		list, err := client.GetReadingList(ctx, metronID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		return continueMetronReadingListWithProgress(ctx, db, covers, *list, progress)
	})
}
