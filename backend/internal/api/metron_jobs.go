package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"
)

type MetronImportJob struct {
	ID        string              `json:"id" doc:"Import job identifier." example:"metron-1"`
	Type      string              `json:"type" doc:"Import type." enum:"comic,readingList,series,character,arc" example:"series"`
	MetronID  int                 `json:"metronId" doc:"Metron resource identifier." example:"123456"`
	Options   MetronImportOptions `json:"options" doc:"Import depth and data-expansion options used by this job."`
	Status    string              `json:"status" doc:"Current job status." enum:"queued,running,canceling,succeeded,failed,canceled" example:"running"`
	Message   string              `json:"message" doc:"Human-readable status message." example:"Importing series from Metron..."`
	Completed int                 `json:"completed" doc:"Completed import units." example:"12"`
	Total     int                 `json:"total" doc:"Total import units, when known." example:"52"`
	StartedAt string              `json:"startedAt" doc:"RFC3339 start timestamp." example:"2026-06-26T12:30:00Z"`
	EndedAt   string              `json:"endedAt,omitempty" doc:"RFC3339 end timestamp, when finished." example:"2026-06-26T12:31:00Z"`
}

type MetronImportJobOutput struct {
	Body MetronImportJob
}

type MetronImportJobListOutput struct {
	Body []MetronImportJob
}

type MetronImportJobEvent struct {
	Job MetronImportJob `json:"job" doc:"Updated Metron import job."`
}

type MetronImportJobInput struct {
	ID string `path:"id" doc:"Import job identifier." example:"metron-1"`
}

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
	id := fmt.Sprintf("metron-%d", s.nextID.Add(1))
	ctx, cancel := context.WithCancel(context.Background())
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

func defaultMetronImportOptions() MetronImportOptions {
	return MetronImportOptions{Mode: "quick"}
}

func resolveMetronImportOptions(options MetronImportOptions) MetronImportOptions {
	force := options.Force
	switch options.Mode {
	case "full":
		return MetronImportOptions{Mode: "full", FullData: normalizeMetronFullData(options.FullData), Force: force}
	default:
		next := defaultMetronImportOptions()
		next.Force = force
		return next
	}
}

func normalizeMetronFullData(values []string) []string {
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(strings.ToLower(value))
		switch value {
		case "comic", "comics", "issue", "issues":
			seen["comics"] = true
		case "series":
			seen["series"] = true
		case "arc", "arcs":
			seen["arcs"] = true
		case "character", "characters":
			seen["characters"] = true
		}
	}
	if len(values) == 0 {
		values = []string{"comics", "series", "arcs", "characters"}
	}
	for _, value := range values {
		add(value)
	}
	if seen["series"] || seen["arcs"] || seen["characters"] {
		seen["comics"] = true
	}
	ordered := []string{}
	for _, value := range []string{"comics", "series", "arcs", "characters"} {
		if seen[value] {
			ordered = append(ordered, value)
		}
	}
	return ordered
}

func (o MetronImportOptions) includesFullData(value string) bool {
	o = resolveMetronImportOptions(o)
	if o.Mode != "full" {
		return false
	}
	for _, item := range o.FullData {
		if item == value {
			return true
		}
	}
	return false
}

func (o MetronImportOptions) includesComics() bool {
	return o.includesFullData("comics")
}

func (o MetronImportOptions) includesSeries() bool {
	return o.includesFullData("series")
}

func (o MetronImportOptions) includesArcs() bool {
	return o.includesFullData("arcs")
}

func (o MetronImportOptions) includesCharacters() bool {
	return o.includesFullData("characters")
}

func (o MetronImportOptions) needsIssueDetail() bool {
	return o.includesComics() || o.includesSeries() || o.includesArcs() || o.includesCharacters()
}

func startMetronComicImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return startMetronComicImportWithOptions(store, db, client, covers, metronID, defaultMetronImportOptions())
}

func startMetronComicImportWithOptions(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithOptions("comic", metronID, options, "Importing comic from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 1, "Checking existing imports...")
		if _, ok, err := existingComicIDByMetronIssueID(ctx, db, metronID); err != nil || ok {
			if ok && options.Mode != "full" && !options.Force {
				progress(1, 1, "Comic already exists.")
				return err
			}
			if err != nil {
				return err
			}
		}

		progress(0, 1, "Fetching comic from Metron...")
		issue, info, err := fetchMetronIssue(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceIssue, metronID); err != nil {
				return err
			}
			if _, ok, err := existingComicIDByMetronIssueID(ctx, db, metronID); err != nil || ok {
				progress(1, 1, "Comic metadata already current.")
				return err
			}
			issue, info, err = fetchMetronIssue(ctx, db, client, metronID, true)
			if err != nil {
				return metronImportError(err)
			}
		}
		_, err = importMetronComicSweep(ctx, db, client, covers, *issue, options, false)
		if err == nil {
			if err := markMetronSynced(ctx, db, metronResourceIssue, metronID, info); err != nil {
				return err
			}
			progress(1, 1, "Comic imported.")
		}
		return err
	})
}

func startMetronReadingListImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return startMetronReadingListImportWithOptions(store, db, client, covers, metronID, defaultMetronImportOptions())
}

func startMetronReadingListImportWithOptions(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithOptions("readingList", metronID, options, "Importing reading list and issues from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Checking existing imports...")
		if _, ok, err := existingReadingOrderIDByMetronID(ctx, db, metronID); err != nil || ok {
			if ok && options.Mode != "full" && !options.Force {
				progress(1, 1, "Reading list already exists.")
				return err
			}
			if err != nil {
				return err
			}
		}

		progress(0, 0, "Fetching reading list from Metron...")
		list, info, err := fetchMetronReadingList(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceReadingList, metronID); err != nil {
				return err
			}
			issues, err := client.GetReadingListIssues(ctx, metronID)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return metronImportError(err)
			}
			list = &metron.ReadingList{ID: metronID, Issues: issues}
		}
		if err := importMetronReadingListWithOptions(ctx, db, client, covers, *list, options.Mode == "full" || options.Force, progress, options); err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceReadingList, metronID, info)
	})
}

func startMetronArcImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return startMetronArcImportWithOptions(store, db, client, covers, metronID, defaultMetronImportOptions())
}

func startMetronArcImportWithOptions(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithOptions("arc", metronID, options, "Importing arc and issues from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Checking existing imports...")
		if _, ok, err := existingArcIDByMetronID(ctx, db, metronID); err != nil || ok {
			if ok && options.Mode != "full" && !options.Force {
				progress(1, 1, "Arc already exists.")
				return err
			}
			if err != nil {
				return err
			}
		}

		progress(0, 0, "Fetching arc from Metron...")
		arc, info, err := fetchMetronArc(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceArc, metronID); err != nil {
				return err
			}
			issues, err := client.GetArcIssues(ctx, metronID)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return metronImportError(err)
			}
			arc = &metron.MetronArc{ID: metronID, Issues: issues}
		}
		if err := importMetronArcWithOptions(ctx, db, client, covers, *arc, options.Mode == "full" || options.Force, progress, options); err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceArc, metronID, info)
	})
}

func startMetronSeriesImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return startMetronSeriesImportWithOptions(store, db, client, covers, metronID, defaultMetronImportOptions())
}

func startMetronSeriesImportWithOptions(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithOptions("series", metronID, options, "Importing series from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching series metadata from Metron...")
		metadata, info, err := fetchMetronSeries(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceSeries, metronID); err != nil {
				return err
			}
		}

		progress(0, 0, "Fetching series issue list from Metron...")
		issues, err := client.GetSeriesIssues(ctx, metronID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		_, err = importMetronSeriesWithProgressOptions(ctx, db, client, covers, issues, progress, options)
		if err != nil {
			return err
		}
		if metadata != nil {
			if err := updateImportedSeriesMetadata(ctx, db, *metadata); err != nil {
				return err
			}
		}
		return markMetronSynced(ctx, db, metronResourceSeries, metronID, info)
	})
}

func startLocalSeriesMetronImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, localID, metronID int) MetronImportJob {
	return store.startWithOptions("series", metronID, defaultMetronImportOptions(), "Importing series from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching series metadata from Metron...")
		metadata, info, err := fetchMetronSeries(ctx, db, client, metronID, false)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceSeries, metronID); err != nil {
				return err
			}
		} else if err := updateSeriesMetronMetadata(ctx, db, localID, *metadata); err != nil {
			return err
		}

		progress(0, 0, "Fetching series issue list from Metron...")
		issues, err := client.GetSeriesIssues(ctx, metronID)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		_, err = importMetronSeriesWithProgress(ctx, db, client, covers, issues, progress)
		if err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceSeries, metronID, info)
	})
}

func startMetronCharacterAppearancesImport(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int) MetronImportJob {
	return startMetronCharacterAppearancesImportWithOptions(store, db, client, covers, metronID, defaultMetronImportOptions())
}

func startMetronCharacterAppearancesImportWithOptions(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithOptions("character", metronID, options, "Importing character from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Preparing character appearance import...")
		return importMetronCharacterAppearancesWithProgressOptions(ctx, db, client, covers, metronID, progress, options)
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

func listMetronImportJobs(store *metronImportJobStore) *MetronImportJobListOutput {
	return &MetronImportJobListOutput{Body: store.list()}
}

func streamMetronImportJobs(ctx context.Context, store *metronImportJobStore, send func(MetronImportJobEvent) error) {
	jobs, unsubscribe := store.subscribe()
	defer unsubscribe()

	for {
		select {
		case <-ctx.Done():
			return
		case job, ok := <-jobs:
			if !ok {
				return
			}
			if err := send(MetronImportJobEvent{Job: job}); err != nil {
				return
			}
		}
	}
}

func getMetronImportJob(store *metronImportJobStore, id string) (*MetronImportJobOutput, error) {
	job, ok := store.get(id)
	if !ok {
		return nil, huma.Error404NotFound("import job not found")
	}
	return &MetronImportJobOutput{Body: job}, nil
}

func deleteMetronImportJob(store *metronImportJobStore, id string) (*struct{}, error) {
	if _, ok, deleted := store.deleteTerminal(id); !ok {
		return nil, huma.Error404NotFound("import job not found")
	} else if !deleted {
		return nil, huma.Error400BadRequest("only finished imports can be dismissed")
	}
	return nil, nil
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
		next = startMetronComicImportWithOptions(store, db, client, covers, job.MetronID, job.Options)
	case "readingList":
		next = startMetronReadingListContinue(store, db, client, covers, job.MetronID, job.Options)
	case "arc":
		next = startMetronArcContinue(store, db, client, covers, job.MetronID, job.Options)
	case "series":
		next = startMetronSeriesImportWithOptions(store, db, client, covers, job.MetronID, job.Options)
	case "character":
		next = startMetronCharacterAppearancesImportWithOptions(store, db, client, covers, job.MetronID, job.Options)
	default:
		return nil, huma.Error400BadRequest("unsupported import type")
	}
	return &MetronImportJobOutput{Body: next}, nil
}

func startMetronArcContinue(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithOptions("arc", metronID, options, "Continuing arc import from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching arc from Metron...")
		arc, info, err := fetchMetronArc(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceArc, metronID); err != nil {
				return err
			}
			issues, err := client.GetArcIssues(ctx, metronID)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return metronImportError(err)
			}
			arc = &metron.MetronArc{ID: metronID, Issues: issues}
		}
		if err := importMetronArcWithOptions(ctx, db, client, covers, *arc, true, progress, options); err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceArc, metronID, info)
	})
}

func startMetronReadingListContinue(store *metronImportJobStore, db *sqlx.DB, client *metron.Client, covers *CoverCache, metronID int, options MetronImportOptions) MetronImportJob {
	options = resolveMetronImportOptions(options)
	return store.startWithOptions("readingList", metronID, options, "Continuing reading list import from Metron...", func(ctx context.Context, progress func(int, int, string)) error {
		progress(0, 0, "Fetching reading list from Metron...")
		list, info, err := fetchMetronReadingList(ctx, db, client, metronID, options.Force)
		if err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			return metronImportError(err)
		}
		if info.NotModified {
			if err := markMetronNotModified(ctx, db, metronResourceReadingList, metronID); err != nil {
				return err
			}
			issues, err := client.GetReadingListIssues(ctx, metronID)
			if err != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				return metronImportError(err)
			}
			list = &metron.ReadingList{ID: metronID, Issues: issues}
		}
		if err := importMetronReadingListWithOptions(ctx, db, client, covers, *list, true, progress, options); err != nil {
			return err
		}
		return markMetronSynced(ctx, db, metronResourceReadingList, metronID, info)
	})
}
