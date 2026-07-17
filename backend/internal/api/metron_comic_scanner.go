package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/jmoiron/sqlx"
)

const (
	metronComicScanSettingsKey = "metron_comic_scan_settings"
	metronComicScanUsageKey    = "metron_comic_scan_usage"
	metronComicScanLastRunKey  = "metron_comic_scan_last_scheduled_date"
)

var weekdayNames = map[string]time.Weekday{
	"sunday": time.Sunday, "monday": time.Monday, "tuesday": time.Tuesday,
	"wednesday": time.Wednesday, "thursday": time.Thursday, "friday": time.Friday,
	"saturday": time.Saturday,
}

var metronComicIncompleteFields = []string{
	"comicVineId",
	"publisher",
	"coverImage",
	"coverDate",
	"description",
}

var metronComicIncompleteConditions = map[string]string{
	"comicVineId": "comic_vine_id IS NULL",
	"publisher":   "TRIM(publisher) = ''",
	"coverImage":  "TRIM(cover_image) = ''",
	"coverDate":   "TRIM(cover_date) = ''",
	"description": "TRIM(description) = ''",
}

type MetronComicScanSettings struct {
	Enabled             bool     `json:"enabled" doc:"Whether automatic and manual incomplete-data scans are enabled."`
	ScanComics          bool     `json:"scanComics" doc:"Whether the comics table is included in scans."`
	Schedule            string   `json:"schedule" enum:"daily,weekly" doc:"Run every day or only on selected weekdays."`
	Weekdays            []string `json:"weekdays,omitempty" doc:"Lowercase weekday names used by a weekly schedule."`
	StartTime           string   `json:"startTime" doc:"Server-local scan start time in HH:MM format." example:"02:00"`
	DailyCallLimit      int      `json:"dailyCallLimit" minimum:"1" doc:"Maximum Metron issue calls shared by all scans during one server-local calendar day." example:"100"`
	MinIntervalSeconds  int      `json:"minIntervalSeconds" minimum:"0" doc:"Minimum seconds between Metron issue calls made by this incomplete-comic scan." example:"20"`
	RecheckCooldownDays int      `json:"recheckCooldownDays" minimum:"0" doc:"Days to wait before re-checking a comic after a Metron lookup, including Comic Vine IDs with no match and fields Metron may not provide. 0 disables the cooldown and rechecks every run." example:"30"`
	IncompleteFields    []string `json:"incompleteFields" doc:"Comic fields whose absence makes a comic eligible for enrichment."`
}

type MetronComicScanStatus struct {
	Settings       MetronComicScanSettings `json:"settings"`
	Running        bool                    `json:"running"`
	StartedAt      string                  `json:"startedAt,omitempty"`
	FinishedAt     string                  `json:"finishedAt,omitempty"`
	StopReason     string                  `json:"stopReason,omitempty"`
	Scanned        int                     `json:"scanned"`
	Updated        int                     `json:"updated"`
	Failed         int                     `json:"failed"`
	CallsUsedToday int                     `json:"callsUsedToday"`
	CallsLeftToday int                     `json:"callsLeftToday"`
	UsageDate      string                  `json:"usageDate"`
}

type MetronComicScanEvent struct {
	Scan MetronComicScanStatus `json:"scan" doc:"Current comic scan settings, quota, and progress."`
}

type metronComicScanUsage struct {
	Date  string `json:"date"`
	Calls int    `json:"calls"`
}

type metronComicScanner struct {
	db     *sqlx.DB
	client *metron.Client
	covers *CoverCache

	mu               sync.Mutex
	status           MetronComicScanStatus
	cancel           context.CancelFunc
	wake             chan struct{}
	shutdown         context.CancelFunc
	nextSubscriberID uint64
	subscribers      map[uint64]chan MetronComicScanStatus
}

func NewMetronComicScanner(db *sqlx.DB, client *metron.Client, covers *CoverCache) *metronComicScanner {
	return &metronComicScanner{db: db, client: client, covers: covers, wake: make(chan struct{}, 1), subscribers: map[uint64]chan MetronComicScanStatus{}}
}

func (s *metronComicScanner) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	s.shutdown = cancel
	go s.scheduleLoop(ctx)
}

func (s *metronComicScanner) Stop() {
	if s.shutdown != nil {
		s.shutdown()
	}
	s.stopScan("server stopped")
}

func defaultMetronComicScanSettings() MetronComicScanSettings {
	return MetronComicScanSettings{
		ScanComics:          true,
		Schedule:            "daily",
		StartTime:           "02:00",
		DailyCallLimit:      100,
		MinIntervalSeconds:  20,
		RecheckCooldownDays: 30,
		IncompleteFields:    append([]string(nil), metronComicIncompleteFields...),
	}
}

func loadMetronComicScanSettings(ctx context.Context, db *sqlx.DB) (MetronComicScanSettings, error) {
	settings := defaultMetronComicScanSettings()
	var value string
	if err := db.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, metronComicScanSettingsKey); err != nil {
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

func validateMetronComicScanSettings(settings *MetronComicScanSettings) error {
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
	selectedFields := map[string]bool{}
	for _, field := range settings.IncompleteFields {
		if _, ok := metronComicIncompleteConditions[field]; !ok {
			return fmt.Errorf("invalid incomplete field %q", field)
		}
		selectedFields[field] = true
	}
	if len(selectedFields) == 0 {
		return errors.New("incompleteFields must contain at least one field")
	}
	settings.IncompleteFields = settings.IncompleteFields[:0]
	for _, field := range metronComicIncompleteFields {
		if selectedFields[field] {
			settings.IncompleteFields = append(settings.IncompleteFields, field)
		}
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
	if !settings.ScanComics {
		return errors.New("scanComics must be enabled while comics are the only supported data type")
	}
	return nil
}

func saveMetronComicScanSettings(ctx context.Context, db *sqlx.DB, settings MetronComicScanSettings) error {
	value, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, metronComicScanSettingsKey, string(value))
	return err
}

func (s *metronComicScanner) scheduleLoop(ctx context.Context) {
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

func (s *metronComicScanner) checkSchedule(ctx context.Context, now time.Time) {
	settings, err := loadMetronComicScanSettings(ctx, s.db)
	if err != nil || !settings.Enabled || !settings.ScanComics || now.Format("15:04") != settings.StartTime {
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
	_ = s.db.GetContext(ctx, &last, `SELECT value FROM app_settings WHERE key = ?`, metronComicScanLastRunKey)
	if last == date {
		return
	}
	if err := s.trigger("scheduled"); err != nil {
		return
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, metronComicScanLastRunKey, date); err != nil {
		return
	}
}

func (s *metronComicScanner) trigger(reason string) error {
	settings, err := loadMetronComicScanSettings(context.Background(), s.db)
	if err != nil {
		return err
	}
	if !settings.Enabled || !settings.ScanComics {
		return errors.New("comic scanning is disabled")
	}
	s.mu.Lock()
	if s.status.Running {
		s.mu.Unlock()
		return errors.New("a comic scan is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.status = MetronComicScanStatus{Settings: settings, Running: true, StartedAt: time.Now().UTC().Format(time.RFC3339), StopReason: reason}
	s.mu.Unlock()
	s.broadcastSnapshot()
	go s.run(ctx, settings)
	return nil
}

func (s *metronComicScanner) stopScan(reason string) bool {
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

func (s *metronComicScanner) run(ctx context.Context, settings MetronComicScanSettings) {
	rows, err := selectIncompleteComics(ctx, s.db, settings, time.Now())
	if err == nil {
		s.setScanned(len(rows))
		var nextRequest time.Time
		interval := time.Duration(settings.MinIntervalSeconds) * time.Second
		claimRequest := func() (bool, error) {
			if waitErr := waitForComicScanInterval(ctx, &nextRequest, interval); waitErr != nil {
				return false, waitErr
			}
			return claimMetronComicScanCall(ctx, s.db, settings.DailyCallLimit, time.Now())
		}
	scanRows:
		for _, row := range rows {
			if ctx.Err() != nil {
				break
			}
			claimed, claimErr := claimRequest()
			if claimErr != nil {
				if ctx.Err() != nil {
					break
				}
				err = claimErr
				break
			}
			if !claimed {
				s.setStopReason("daily quota used")
				break
			}

			var issue *metron.Issue
			var fetchErr error
			if row.MetronID.Valid {
				issue, fetchErr = s.client.GetIssue(ctx, int(row.MetronID.Int64))
			} else {
				matches, searchErr := s.client.SearchIssuesByComicVineID(ctx, int(row.ComicVineID.Int64))
				if searchErr != nil {
					fetchErr = searchErr
				} else if len(matches) == 0 {
					if markIncompleteComicChecked(ctx, s.db, row.ID, time.Now()) != nil {
						s.incrementFailed()
					}
					continue
				} else if len(matches) > 1 {
					fetchErr = fmt.Errorf("Comic Vine ID %d returned %d Metron issues", row.ComicVineID.Int64, len(matches))
				} else {
					claimed, claimErr = claimRequest()
					if claimErr != nil {
						if ctx.Err() == nil {
							err = claimErr
						}
						break scanRows
					}
					if !claimed {
						s.setStopReason("daily quota used")
						break scanRows
					}
					issue, fetchErr = s.client.GetIssue(ctx, matches[0].ID)
				}
			}
			if fetchErr != nil {
				if ctx.Err() != nil {
					break
				}
				s.incrementFailed()
				continue
			}
			if enrichIncompleteComicFromMetron(ctx, s.db, s.covers, row.ID, *issue) != nil {
				s.incrementFailed()
			} else {
				s.incrementUpdated()
			}
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

type incompleteComicRow struct {
	ID          int           `db:"id"`
	MetronID    sql.NullInt64 `db:"metron_issue_id"`
	ComicVineID sql.NullInt64 `db:"comic_vine_id"`
}

func selectIncompleteComics(ctx context.Context, db *sqlx.DB, settings MetronComicScanSettings, now time.Time) ([]incompleteComicRow, error) {
	conditions := make([]string, 0, len(settings.IncompleteFields))
	for _, field := range settings.IncompleteFields {
		if condition, ok := metronComicIncompleteConditions[field]; ok {
			conditions = append(conditions, condition)
		}
	}
	if len(conditions) == 0 {
		return []incompleteComicRow{}, nil
	}

	query := `SELECT id, metron_issue_id, comic_vine_id FROM comics WHERE (metron_issue_id IS NOT NULL OR comic_vine_id IS NOT NULL) AND (` + strings.Join(conditions, " OR ") + `)`
	args := []any{}
	if settings.RecheckCooldownDays > 0 {
		cutoff := now.Add(-time.Duration(settings.RecheckCooldownDays) * 24 * time.Hour).UTC().Format(time.RFC3339)
		query += ` AND (metron_synced_at = '' OR metron_synced_at <= ?)`
		args = append(args, cutoff)
	}
	query += ` ORDER BY id`

	rows := []incompleteComicRow{}
	if err := db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func waitForComicScanInterval(ctx context.Context, nextRequest *time.Time, interval time.Duration) error {
	if interval <= 0 {
		return nil
	}
	if wait := time.Until(*nextRequest); wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
		}
	}
	*nextRequest = time.Now().Add(interval)
	return nil
}

func enrichIncompleteComicFromMetron(ctx context.Context, db *sqlx.DB, covers *CoverCache, comicID int, issue metron.Issue) error {
	cover := issue.CoverImage
	if cover != "" {
		var current string
		if err := db.GetContext(ctx, &current, `SELECT cover_image FROM comics WHERE id = ?`, comicID); err != nil {
			return err
		}
		if strings.TrimSpace(current) == "" {
			var err error
			cover, err = localCoverURL(ctx, covers, cover)
			if err != nil {
				return err
			}
		} else {
			cover = current
		}
	}
	syncedAt := time.Now().UTC().Format(time.RFC3339)
	_, err := db.ExecContext(ctx, `UPDATE comics SET metron_issue_id = COALESCE(metron_issue_id, ?), comic_vine_id = COALESCE(comic_vine_id, ?), publisher = CASE WHEN TRIM(publisher) = '' THEN ? ELSE publisher END, cover_date = CASE WHEN TRIM(cover_date) = '' THEN ? ELSE cover_date END, cover_image = CASE WHEN TRIM(cover_image) = '' THEN ? ELSE cover_image END, description = CASE WHEN TRIM(description) = '' THEN ? ELSE description END, metron_synced_at = ? WHERE id = ?`, nullablePositiveID(issue.ID), nullablePositiveID(issue.ComicVineID), issue.Publisher, issue.CoverDate, cover, issue.Description, syncedAt, comicID)
	if err != nil {
		return err
	}
	// The issue response already contains lightweight arc and character data. A
	// nil client prevents these helpers from making any detail requests.
	options := MetronImportOptions{Mode: "basic"}
	if err := syncMetronIssueArcsWithOptions(ctx, db, nil, comicID, issue, options); err != nil {
		return err
	}
	return syncMetronIssueCharactersWithOptions(ctx, db, nil, covers, comicID, issue, options)
}

func markIncompleteComicChecked(ctx context.Context, db *sqlx.DB, comicID int, checkedAt time.Time) error {
	_, err := db.ExecContext(ctx, `UPDATE comics SET metron_synced_at = ? WHERE id = ?`, checkedAt.UTC().Format(time.RFC3339), comicID)
	return err
}

func claimMetronComicScanCall(ctx context.Context, db *sqlx.DB, limit int, now time.Time) (bool, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	date := now.Format("2006-01-02")
	usage := metronComicScanUsage{Date: date}
	var value string
	if err := tx.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, metronComicScanUsageKey); err == nil {
		_ = json.Unmarshal([]byte(value), &usage)
	} else if err != sql.ErrNoRows {
		return false, err
	}
	if usage.Date != date {
		usage = metronComicScanUsage{Date: date}
	}
	if usage.Calls >= limit {
		return false, nil
	}
	usage.Calls++
	encoded, _ := json.Marshal(usage)
	if _, err := tx.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, metronComicScanUsageKey, string(encoded)); err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

func currentMetronComicScanUsage(ctx context.Context, db *sqlx.DB, now time.Time) metronComicScanUsage {
	usage := metronComicScanUsage{Date: now.Format("2006-01-02")}
	var value string
	if db.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, metronComicScanUsageKey) == nil {
		_ = json.Unmarshal([]byte(value), &usage)
	}
	if usage.Date != now.Format("2006-01-02") {
		return metronComicScanUsage{Date: now.Format("2006-01-02")}
	}
	return usage
}

func (s *metronComicScanner) snapshot(ctx context.Context) MetronComicScanStatus {
	settings, _ := loadMetronComicScanSettings(ctx, s.db)
	usage := currentMetronComicScanUsage(ctx, s.db, time.Now())
	s.mu.Lock()
	status := s.status
	s.mu.Unlock()
	status.Settings = settings
	status.CallsUsedToday = usage.Calls
	status.CallsLeftToday = max(0, settings.DailyCallLimit-usage.Calls)
	status.UsageDate = usage.Date
	return status
}

func (s *metronComicScanner) setScanned(count int) {
	s.mu.Lock()
	s.status.Scanned = count
	s.mu.Unlock()
	s.broadcastSnapshot()
}
func (s *metronComicScanner) incrementUpdated() {
	s.mu.Lock()
	s.status.Updated++
	s.mu.Unlock()
	s.broadcastSnapshot()
}
func (s *metronComicScanner) incrementFailed() {
	s.mu.Lock()
	s.status.Failed++
	s.mu.Unlock()
	s.broadcastSnapshot()
}
func (s *metronComicScanner) setStopReason(v string) {
	s.mu.Lock()
	s.status.StopReason = v
	s.mu.Unlock()
	s.broadcastSnapshot()
}

func (s *metronComicScanner) subscribe(ctx context.Context) (<-chan MetronComicScanStatus, func()) {
	s.mu.Lock()
	s.nextSubscriberID++
	id := s.nextSubscriberID
	ch := make(chan MetronComicScanStatus, 16)
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

func (s *metronComicScanner) broadcastSnapshot() {
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

func streamMetronComicScan(ctx context.Context, scanner *metronComicScanner, send func(MetronComicScanEvent) error) {
	updates, unsubscribe := scanner.subscribe(ctx)
	defer unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case status, ok := <-updates:
			if !ok || send(MetronComicScanEvent{Scan: status}) != nil {
				return
			}
		}
	}
}

type MetronComicScanStatusOutput struct{ Body MetronComicScanStatus }
type UpdateMetronComicScanSettingsInput struct{ Body MetronComicScanSettings }

func registerMetronComicScannerRoutes(api huma.API, db *sqlx.DB, scanner *metronComicScanner) {
	huma.Register(api, huma.Operation{OperationID: "getMetronComicScan", Tags: []string{tagMetron}, Summary: "Get comic scan settings and status", Method: http.MethodGet, Path: "/metron/scans/comics", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		return &MetronComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
	sse.Register(api, huma.Operation{OperationID: "streamMetronComicScan", Tags: []string{tagMetron}, Summary: "Stream comic scan status", Description: "Streams an initial snapshot and live comic scan settings, quota, and progress updates.", Method: http.MethodGet, Path: "/metron/scans/comics/events", Errors: []int{401, 403, 500}}, map[string]any{"scan": MetronComicScanEvent{}}, func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return
		}
		streamMetronComicScan(ctx, scanner, func(event MetronComicScanEvent) error { return send.Data(event) })
	})
	huma.Register(api, huma.Operation{OperationID: "updateMetronComicScan", Tags: []string{tagMetron}, Summary: "Update comic scan settings", Method: http.MethodPut, Path: "/metron/scans/comics", Errors: []int{400, 401, 403, 500}}, func(ctx context.Context, input *UpdateMetronComicScanSettingsInput) (*MetronComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := validateMetronComicScanSettings(&input.Body); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := saveMetronComicScanSettings(ctx, db, input.Body); err != nil {
			return nil, huma.Error500InternalServerError("failed to save comic scan settings")
		}
		select {
		case scanner.wake <- struct{}{}:
		default:
		}
		scanner.broadcastSnapshot()
		return &MetronComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "triggerMetronComicScan", Tags: []string{tagMetron}, Summary: "Trigger comic scan", Method: http.MethodPost, Path: "/metron/scans/comics/trigger", DefaultStatus: http.StatusAccepted, Errors: []int{400, 401, 403, 409, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := scanner.trigger("manual"); err != nil {
			if strings.Contains(err.Error(), "already running") {
				return nil, huma.Error409Conflict(err.Error())
			}
			return nil, huma.Error400BadRequest(err.Error())
		}
		return &MetronComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "stopMetronComicScan", Tags: []string{tagMetron}, Summary: "Stop comic scan", Method: http.MethodPost, Path: "/metron/scans/comics/stop", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		scanner.stopScan("stopped by admin")
		return &MetronComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
}
