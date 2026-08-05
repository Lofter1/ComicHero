package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Lofter1/ComicHero/backend/comicvine"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/jmoiron/sqlx"
)

const (
	comicVineComicScanSettingsKey = "comicvine_comic_scan_settings"
	comicVineComicScanUsageKey    = "comicvine_comic_scan_usage"
	comicVineComicScanLastRunKey  = "comicvine_comic_scan_last_scheduled_date"
)

// comicVineComicIncompleteFields intentionally excludes an ID field: unlike
// Metron maintenance, Comic Vine maintenance never looks a comic up by
// series/issue - it only enriches comics that already carry a comic_vine_id,
// so there's nothing analogous to "comicVineId" to flag as missing.
var comicVineComicIncompleteFields = []string{
	"publisher",
	"coverImage",
	"coverDate",
	"description",
}

var comicVineComicIncompleteConditions = map[string]string{
	"publisher":   "TRIM(publisher) = ''",
	"coverImage":  "TRIM(cover_image) = ''",
	"coverDate":   "TRIM(cover_date) = ''",
	"description": "TRIM(description) = ''",
}

type ComicVineComicScanSettings struct {
	Enabled             bool     `json:"enabled" doc:"Whether automatic and manual incomplete-data scans are enabled."`
	Schedule            string   `json:"schedule" enum:"daily,weekly" doc:"Run every day or only on selected weekdays."`
	Weekdays            []string `json:"weekdays,omitempty" doc:"Lowercase weekday names used by a weekly schedule."`
	StartTime           string   `json:"startTime" doc:"Server-local scan start time in HH:MM format." example:"02:30"`
	DailyCallLimit      int      `json:"dailyCallLimit" minimum:"1" doc:"Maximum Comic Vine calls made by this maintenance scan during one server-local calendar day." example:"100"`
	MinIntervalSeconds  int      `json:"minIntervalSeconds" minimum:"0" doc:"Minimum seconds between Comic Vine calls made by this maintenance scan." example:"20"`
	RecheckCooldownDays int      `json:"recheckCooldownDays" minimum:"0" doc:"Days to wait before re-checking an incomplete comic after a Comic Vine lookup. 0 disables the cooldown and rechecks every run." example:"30"`
	IncompleteFields    []string `json:"incompleteFields" doc:"Comic fields whose absence makes a comic eligible for enrichment."`
}

type ComicVineComicScanStatus struct {
	Settings         ComicVineComicScanSettings `json:"settings"`
	APIKeyConfigured bool                       `json:"apiKeyConfigured" doc:"Whether a Comic Vine API key is configured on the server. Comic Vine maintenance cannot run without one."`
	Running          bool                       `json:"running"`
	StartedAt        string                     `json:"startedAt,omitempty"`
	FinishedAt       string                     `json:"finishedAt,omitempty"`
	StopReason       string                     `json:"stopReason,omitempty"`
	Scanned          int                        `json:"scanned"`
	Updated          int                        `json:"updated"`
	Failed           int                        `json:"failed"`
	LastError        string                     `json:"lastError,omitempty"`
	CallsUsedToday   int                        `json:"callsUsedToday"`
	CallsLeftToday   int                        `json:"callsLeftToday"`
	UsageDate        string                     `json:"usageDate"`
}

type ComicVineComicScanEvent struct {
	Scan ComicVineComicScanStatus `json:"scan" doc:"Current Comic Vine maintenance settings, quota, and progress."`
}

type comicVineComicScanUsage struct {
	Date  string `json:"date"`
	Calls int    `json:"calls"`
}

type comicVineComicScanner struct {
	db     *sqlx.DB
	client *comicvine.Client
	covers *CoverCache

	mu               sync.Mutex
	status           ComicVineComicScanStatus
	cancel           context.CancelFunc
	wake             chan struct{}
	shutdown         context.CancelFunc
	nextSubscriberID uint64
	subscribers      map[uint64]chan ComicVineComicScanStatus
}

func NewComicVineComicScanner(db *sqlx.DB, client *comicvine.Client, covers *CoverCache) *comicVineComicScanner {
	return &comicVineComicScanner{
		db: db, client: client, covers: covers,
		wake:        make(chan struct{}, 1),
		subscribers: map[uint64]chan ComicVineComicScanStatus{},
	}
}

func (s *comicVineComicScanner) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.shutdown = cancel
	go s.scheduleLoop(ctx)
}

func (s *comicVineComicScanner) Stop() {
	if s.shutdown != nil {
		s.shutdown()
	}
	s.stopScan("server stopped")
}

func defaultComicVineComicScanSettings() ComicVineComicScanSettings {
	return ComicVineComicScanSettings{
		Schedule:            "daily",
		StartTime:           "02:30",
		DailyCallLimit:      100,
		MinIntervalSeconds:  20,
		RecheckCooldownDays: 30,
		IncompleteFields:    append([]string(nil), comicVineComicIncompleteFields...),
	}
}

func loadComicVineComicScanSettings(ctx context.Context, db *sqlx.DB) (ComicVineComicScanSettings, error) {
	settings := defaultComicVineComicScanSettings()
	var value string
	if err := db.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, comicVineComicScanSettingsKey); err != nil {
		if err == sql.ErrNoRows {
			return settings, nil
		}
		return settings, err
	}
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return settings, err
	}
	return settings, nil
}

func validateComicVineComicScanSettings(settings *ComicVineComicScanSettings) error {
	settings.Schedule = strings.ToLower(strings.TrimSpace(settings.Schedule))
	if settings.Schedule != "daily" && settings.Schedule != "weekly" {
		return errors.New("schedule must be daily or weekly")
	}
	if _, err := time.Parse("15:04", settings.StartTime); err != nil {
		return errors.New("startTime must use HH:MM")
	}
	if settings.DailyCallLimit < 1 {
		return errors.New("dailyCallLimit must be at least 1")
	}
	if settings.MinIntervalSeconds < 0 {
		return errors.New("minIntervalSeconds cannot be negative")
	}
	if settings.RecheckCooldownDays < 0 {
		return errors.New("recheckCooldownDays cannot be negative")
	}
	var err error
	if settings.IncompleteFields, err = normalizeIncompleteFields(
		settings.IncompleteFields,
		comicVineComicIncompleteFields,
		comicVineComicIncompleteConditions,
		true,
		"incompleteFields",
	); err != nil {
		return err
	}
	seen := map[string]bool{}
	weekdays := make([]string, 0, len(settings.Weekdays))
	for _, day := range settings.Weekdays {
		day = strings.ToLower(strings.TrimSpace(day))
		if _, ok := weekdayNames[day]; !ok {
			return fmt.Errorf("invalid weekday %q", day)
		}
		if !seen[day] {
			seen[day] = true
			weekdays = append(weekdays, day)
		}
	}
	sort.Strings(weekdays)
	settings.Weekdays = weekdays
	if settings.Schedule == "weekly" && len(weekdays) == 0 {
		return errors.New("weekly schedules need at least one weekday")
	}
	return nil
}

func saveComicVineComicScanSettings(ctx context.Context, db *sqlx.DB, settings ComicVineComicScanSettings) error {
	value, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, comicVineComicScanSettingsKey, string(value))
	return err
}

func (s *comicVineComicScanner) scheduleLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkSchedule(ctx, time.Now())
		case <-s.wake:
			s.checkSchedule(ctx, time.Now())
		}
	}
}

func (s *comicVineComicScanner) checkSchedule(ctx context.Context, now time.Time) {
	settings, err := loadComicVineComicScanSettings(ctx, s.db)
	if err != nil || !settings.Enabled || now.Format("15:04") != settings.StartTime {
		return
	}
	if settings.Schedule == "weekly" {
		wanted := false
		for _, day := range settings.Weekdays {
			if weekdayNames[day] == now.Weekday() {
				wanted = true
				break
			}
		}
		if !wanted {
			return
		}
	}
	date := now.Format("2006-01-02")
	var last string
	_ = s.db.GetContext(ctx, &last, `SELECT value FROM app_settings WHERE key = ?`, comicVineComicScanLastRunKey)
	if last == date {
		return
	}
	if err := s.trigger("scheduled"); err != nil {
		return
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, comicVineComicScanLastRunKey, date); err != nil {
		return
	}
}

func (s *comicVineComicScanner) trigger(reason string) error {
	if s.client == nil || !s.client.HasAPIKey() {
		return errors.New("no Comic Vine API key is configured")
	}
	settings, err := loadComicVineComicScanSettings(context.Background(), s.db)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return errors.New("comic vine maintenance is disabled")
	}
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return errors.New("a comic scan is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.status = ComicVineComicScanStatus{Settings: settings, Running: true, StartedAt: time.Now().UTC().Format(time.RFC3339), StopReason: reason}
	s.mu.Unlock()
	s.broadcastSnapshot()
	go s.run(ctx, settings)
	return nil
}

func (s *comicVineComicScanner) stopScan(reason string) bool {
	s.mu.Lock()
	if !s.status.Running || s.cancel == nil {
		s.mu.Unlock()
		return false
	}
	s.status.StopReason = reason
	s.cancel()
	s.mu.Unlock()
	s.broadcastSnapshot()
	return true
}

func (s *comicVineComicScanner) run(ctx context.Context, settings ComicVineComicScanSettings) {
	comics, err := selectIncompleteComicVineComics(ctx, s.db, settings, time.Now())
	if err == nil {
		s.setScanned(len(comics))
		budget := comicVineMaintenanceBudget{
			db:       s.db,
			limit:    settings.DailyCallLimit,
			interval: time.Duration(settings.MinIntervalSeconds) * time.Second,
		}
		err = s.scanIncompleteComicVineComics(ctx, comics, settings, &budget)
		if errors.Is(err, errComicVineMaintenanceQuotaUsed) {
			s.setStopReason("daily quota used")
			err = nil
		}
	}
	s.mu.Lock()
	s.status.Running = false
	s.status.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if err != nil {
		s.status.StopReason = err.Error()
	} else if ctx.Err() != nil && s.status.StopReason == "" {
		s.status.StopReason = "stopped"
	} else if s.status.StopReason == "scheduled" || s.status.StopReason == "manual" {
		s.status.StopReason = "complete"
	}
	s.cancel = nil
	s.mu.Unlock()
	s.broadcastSnapshot()
}

var errComicVineMaintenanceQuotaUsed = errors.New("daily Comic Vine maintenance quota used")

type comicVineMaintenanceBudget struct {
	db          *sqlx.DB
	limit       int
	interval    time.Duration
	nextRequest time.Time
}

func (b *comicVineMaintenanceBudget) claim(ctx context.Context) error {
	if err := waitForComicScanInterval(ctx, &b.nextRequest, b.interval); err != nil {
		return err
	}
	claimed, err := claimComicVineComicScanCall(ctx, b.db, b.limit, time.Now())
	if err != nil {
		return err
	}
	if !claimed {
		return errComicVineMaintenanceQuotaUsed
	}
	return nil
}

type incompleteComicVineComicRow struct {
	ID          int `db:"id"`
	ComicVineID int `db:"comic_vine_id"`
}

func selectIncompleteComicVineComics(ctx context.Context, db *sqlx.DB, settings ComicVineComicScanSettings, now time.Time) ([]incompleteComicVineComicRow, error) {
	conditions := make([]string, 0, len(settings.IncompleteFields))
	for _, field := range settings.IncompleteFields {
		if condition, ok := comicVineComicIncompleteConditions[field]; ok {
			conditions = append(conditions, condition)
		}
	}
	if len(conditions) == 0 {
		return []incompleteComicVineComicRow{}, nil
	}

	query := `SELECT id, comic_vine_id FROM comics WHERE comic_vine_id IS NOT NULL AND (` + strings.Join(conditions, " OR ") + `)`
	args := []any{}
	if settings.RecheckCooldownDays > 0 {
		cutoff := now.Add(-time.Duration(settings.RecheckCooldownDays) * 24 * time.Hour).UTC().Format(time.RFC3339)
		query += ` AND (comic_vine_synced_at = '' OR comic_vine_synced_at <= ?)`
		args = append(args, cutoff)
	}
	query += ` ORDER BY id`

	rows := []incompleteComicVineComicRow{}
	if err := db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *comicVineComicScanner) scanIncompleteComicVineComics(ctx context.Context, rows []incompleteComicVineComicRow, settings ComicVineComicScanSettings, budget *comicVineMaintenanceBudget) error {
	wantPublisher := false
	for _, field := range settings.IncompleteFields {
		if field == "publisher" {
			wantPublisher = true
		}
	}

	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := budget.claim(ctx); err != nil {
			return err
		}

		issue, fetchErr := s.client.Issues().GetByID(ctx, row.ComicVineID, []string{"cover_date", "description", "image", "volume"})
		if fetchErr != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
			s.recordFailure("comic", row.ID, fmt.Errorf("fetch Comic Vine issue: %w", fetchErr))
			continue
		}

		publisher := ""
		if wantPublisher && issue.Volume != nil && issue.Volume.ID > 0 {
			if err := budget.claim(ctx); err != nil {
				return err
			}
			volume, volumeErr := s.client.Volumes().GetByID(ctx, issue.Volume.ID, []string{"publisher"})
			if volumeErr != nil {
				if err := ctx.Err(); err != nil {
					return err
				}
				s.recordFailure("comic", row.ID, fmt.Errorf("fetch Comic Vine volume: %w", volumeErr))
			} else {
				publisher = volume.Publisher.Name
			}
		}

		if enrichErr := enrichIncompleteComicFromComicVine(ctx, s.db, s.covers, row.ID, *issue, publisher); enrichErr != nil {
			s.recordFailure("comic", row.ID, enrichErr)
		} else {
			s.incrementUpdated()
		}
	}
	return nil
}

func enrichIncompleteComicFromComicVine(ctx context.Context, db *sqlx.DB, covers *CoverCache, comicID int, issue comicvine.Issue, publisher string) error {
	cover := issue.Image.OriginalURL
	if cover == "" {
		cover = issue.Image.MediumURL
	}
	if cover != "" {
		var current string
		if err := db.GetContext(ctx, &current, `SELECT cover_image FROM comics WHERE id = ?`, comicID); err != nil {
			return err
		}
		if strings.TrimSpace(current) == "" {
			var err error
			cover, err = localCoverURL(ctx, covers, cover)
			if err != nil {
				return fmt.Errorf("cache cover: %w", err)
			}
		} else {
			cover = current
		}
	}
	_, err := db.ExecContext(ctx, `
		UPDATE comics SET
			publisher = CASE WHEN TRIM(publisher) = '' THEN ? ELSE publisher END,
			cover_date = CASE WHEN TRIM(cover_date) = '' THEN ? ELSE cover_date END,
			cover_image = CASE WHEN TRIM(cover_image) = '' THEN ? ELSE cover_image END,
			description = CASE WHEN TRIM(description) = '' THEN ? ELSE description END
		WHERE id = ?
	`, publisher, issue.CoverDate, cover, issue.Description, comicID)
	if err != nil {
		return fmt.Errorf("update comic metadata: %w", err)
	}
	if err := markIncompleteComicVineComicChecked(ctx, db, comicID, time.Now()); err != nil {
		return fmt.Errorf("mark comic as synced: %w", err)
	}
	return nil
}

func markIncompleteComicVineComicChecked(ctx context.Context, db *sqlx.DB, comicID int, checkedAt time.Time) error {
	_, err := db.ExecContext(ctx, `UPDATE comics SET comic_vine_synced_at = ? WHERE id = ?`, checkedAt.UTC().Format(time.RFC3339), comicID)
	return err
}

func claimComicVineComicScanCall(ctx context.Context, db *sqlx.DB, limit int, now time.Time) (bool, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	date := now.Format("2006-01-02")
	usage := comicVineComicScanUsage{Date: date}
	var value string
	if err := tx.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, comicVineComicScanUsageKey); err == nil {
		_ = json.Unmarshal([]byte(value), &usage)
	} else if err != sql.ErrNoRows {
		return false, err
	}
	if usage.Date != date {
		usage = comicVineComicScanUsage{Date: date}
	}
	if usage.Calls >= limit {
		return false, nil
	}
	usage.Calls++
	encoded, _ := json.Marshal(usage)
	if _, err := tx.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, comicVineComicScanUsageKey, string(encoded)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func currentComicVineComicScanUsage(ctx context.Context, db *sqlx.DB, now time.Time) comicVineComicScanUsage {
	usage := comicVineComicScanUsage{Date: now.Format("2006-01-02")}
	var value string
	if db.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, comicVineComicScanUsageKey) == nil {
		_ = json.Unmarshal([]byte(value), &usage)
	}
	if usage.Date != now.Format("2006-01-02") {
		return comicVineComicScanUsage{Date: now.Format("2006-01-02")}
	}
	return usage
}

func (s *comicVineComicScanner) snapshot(ctx context.Context) ComicVineComicScanStatus {
	settings, _ := loadComicVineComicScanSettings(ctx, s.db)
	usage := currentComicVineComicScanUsage(ctx, s.db, time.Now())
	s.mu.Lock()
	status := s.status
	s.mu.Unlock()
	status.Settings = settings
	status.APIKeyConfigured = s.client != nil && s.client.HasAPIKey()
	status.CallsUsedToday = usage.Calls
	status.CallsLeftToday = max(0, settings.DailyCallLimit-usage.Calls)
	status.UsageDate = usage.Date
	return status
}

func (s *comicVineComicScanner) setScanned(count int) {
	s.mu.Lock()
	s.status.Scanned = count
	s.mu.Unlock()
	s.broadcastSnapshot()
}
func (s *comicVineComicScanner) incrementUpdated() {
	s.mu.Lock()
	s.status.Updated++
	s.mu.Unlock()
	s.broadcastSnapshot()
}
func (s *comicVineComicScanner) recordFailure(resourceType string, localID int, err error) {
	message := fmt.Sprintf("%s %d: %v", resourceType, localID, err)
	log.Printf("Comic Vine maintenance failed: %s", message)
	s.mu.Lock()
	s.status.Failed++
	s.status.LastError = message
	s.mu.Unlock()
	s.broadcastSnapshot()
}
func (s *comicVineComicScanner) setStopReason(v string) {
	s.mu.Lock()
	s.status.StopReason = v
	s.mu.Unlock()
	s.broadcastSnapshot()
}

func (s *comicVineComicScanner) subscribe(ctx context.Context) (<-chan ComicVineComicScanStatus, func()) {
	s.mu.Lock()
	s.nextSubscriberID++
	id := s.nextSubscriberID
	ch := make(chan ComicVineComicScanStatus, 16)
	s.subscribers[id] = ch
	s.mu.Unlock()
	ch <- s.snapshot(ctx)
	return ch, func() {
		s.mu.Lock()
		if current, ok := s.subscribers[id]; ok {
			delete(s.subscribers, id)
			close(current)
		}
		s.mu.Unlock()
	}
}

func (s *comicVineComicScanner) broadcastSnapshot() {
	status := s.snapshot(context.Background())
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, ch := range s.subscribers {
		select {
		case ch <- status:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- status:
			default:
			}
		}
	}
}

func streamComicVineComicScan(ctx context.Context, scanner *comicVineComicScanner, send func(ComicVineComicScanEvent) error) {
	updates, unsubscribe := scanner.subscribe(ctx)
	defer unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case status, ok := <-updates:
			if !ok || send(ComicVineComicScanEvent{Scan: status}) != nil {
				return
			}
		}
	}
}

type ComicVineComicScanStatusOutput struct{ Body ComicVineComicScanStatus }
type UpdateComicVineComicScanSettingsInput struct{ Body ComicVineComicScanSettings }

func registerComicVineComicScannerRoutes(api huma.API, db *sqlx.DB, scanner *comicVineComicScanner) {
	huma.Register(api, huma.Operation{OperationID: "getComicVineComicScan", Tags: []string{tagComicVine}, Summary: "Get Comic Vine maintenance settings and status", Method: http.MethodGet, Path: "/comicvine/scans/comics", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*ComicVineComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		return &ComicVineComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
	sse.Register(api, huma.Operation{OperationID: "streamComicVineComicScan", Tags: []string{tagComicVine}, Summary: "Stream Comic Vine maintenance status", Description: "Streams an initial snapshot and live Comic Vine maintenance settings, quota, and progress updates.", Method: http.MethodGet, Path: "/comicvine/scans/comics/events", Errors: []int{401, 403, 500}}, map[string]any{"scan": ComicVineComicScanEvent{}}, func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return
		}
		streamComicVineComicScan(ctx, scanner, func(event ComicVineComicScanEvent) error { return send.Data(event) })
	})
	huma.Register(api, huma.Operation{OperationID: "updateComicVineComicScan", Tags: []string{tagComicVine}, Summary: "Update Comic Vine maintenance settings", Method: http.MethodPut, Path: "/comicvine/scans/comics", Errors: []int{400, 401, 403, 500}}, func(ctx context.Context, input *UpdateComicVineComicScanSettingsInput) (*ComicVineComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := validateComicVineComicScanSettings(&input.Body); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := saveComicVineComicScanSettings(ctx, db, input.Body); err != nil {
			return nil, huma.Error500InternalServerError("failed to save Comic Vine maintenance settings")
		}
		select {
		case scanner.wake <- struct{}{}:
		default:
		}
		scanner.broadcastSnapshot()
		return &ComicVineComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "triggerComicVineComicScan", Tags: []string{tagComicVine}, Summary: "Trigger Comic Vine maintenance", Method: http.MethodPost, Path: "/comicvine/scans/comics/trigger", DefaultStatus: http.StatusAccepted, Errors: []int{400, 401, 403, 409, 500}}, func(ctx context.Context, _ *struct{}) (*ComicVineComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := scanner.trigger("manual"); err != nil {
			if strings.Contains(err.Error(), "already running") {
				return nil, huma.Error409Conflict(err.Error())
			}
			return nil, huma.Error400BadRequest(err.Error())
		}
		return &ComicVineComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "stopComicVineComicScan", Tags: []string{tagComicVine}, Summary: "Stop Comic Vine maintenance", Method: http.MethodPost, Path: "/comicvine/scans/comics/stop", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*ComicVineComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		scanner.stopScan("stopped by admin")
		return &ComicVineComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
}

// RegisterComicVineRoutes wires up the Comic Vine maintenance endpoints.
func RegisterComicVineRoutes(api huma.API, db *sqlx.DB, scanner *comicVineComicScanner) {
	if scanner == nil {
		return
	}
	registerComicVineComicScannerRoutes(api, db, scanner)
}
