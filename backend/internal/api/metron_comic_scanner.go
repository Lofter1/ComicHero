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

var metronCharacterIncompleteFields = []string{
	"description",
	"image",
	"aliases",
}

var metronCharacterIncompleteConditions = map[string]string{
	"description": "TRIM(ch.description) = ''",
	"image":       "TRIM(ch.image) = ''",
	"aliases":     "NOT EXISTS (SELECT 1 FROM character_aliases ca WHERE ca.character_id = ch.id)",
}

var metronSeriesIncompleteFields = []string{
	"publisher",
	"seriesYear",
	"volume",
	"yearEnd",
	"issueCount",
	"description",
}

var metronSeriesIncompleteConditions = map[string]string{
	"publisher":   "TRIM(s.publisher) = ''",
	"seriesYear":  "s.series_year = 0",
	"volume":      "s.volume = 0",
	"yearEnd":     "s.year_end = 0",
	"issueCount":  "s.issue_count = 0",
	"description": "TRIM(s.description) = ''",
}

var metronArcIncompleteFields = []string{
	"description",
	"image",
}

var metronArcIncompleteConditions = map[string]string{
	"description": "TRIM(a.description) = ''",
	"image":       "TRIM(a.image) = ''",
}

var metronMaintenanceResourceOrder = []string{
	"comics",
	"characters",
	"series",
	"arcs",
}

type MetronComicScanSettings struct {
	Enabled                   bool     `json:"enabled" doc:"Whether automatic and manual incomplete-data scans are enabled."`
	ScanComics                bool     `json:"scanComics" doc:"Whether comics are included in scans."`
	ScanCharacters            bool     `json:"scanCharacters" doc:"Whether Metron-linked characters are included in scans."`
	ScanSeries                bool     `json:"scanSeries" doc:"Whether Metron-linked series are included in scans."`
	ScanArcs                  bool     `json:"scanArcs" doc:"Whether Metron-linked story arcs are included in scans."`
	PullCharacterComics       bool     `json:"pullCharacterComics" doc:"Whether character maintenance also pulls and imports the complete Metron appearance list."`
	PullSeriesComics          bool     `json:"pullSeriesComics" doc:"Whether series maintenance also pulls and imports the complete Metron issue list."`
	PullArcComics             bool     `json:"pullArcComics" doc:"Whether story-arc maintenance also pulls and imports the complete Metron issue list."`
	Schedule                  string   `json:"schedule" enum:"daily,weekly" doc:"Run every day or only on selected weekdays."`
	Weekdays                  []string `json:"weekdays,omitempty" doc:"Lowercase weekday names used by a weekly schedule."`
	StartTime                 string   `json:"startTime" doc:"Server-local scan start time in HH:MM format." example:"02:00"`
	DailyCallLimit            int      `json:"dailyCallLimit" minimum:"1" doc:"Maximum Metron calls shared by all maintenance resource types during one server-local calendar day." example:"100"`
	MinIntervalSeconds        int      `json:"minIntervalSeconds" minimum:"0" doc:"Minimum seconds between Metron calls made by this maintenance scan." example:"20"`
	RecheckCooldownDays       int      `json:"recheckCooldownDays" minimum:"0" doc:"Days to wait before re-checking an incomplete record after a Metron lookup. 0 disables the cooldown and rechecks every run." example:"30"`
	IncompleteFields          []string `json:"incompleteFields" doc:"Comic fields whose absence makes a comic eligible for enrichment."`
	CharacterIncompleteFields []string `json:"characterIncompleteFields" doc:"Character fields whose absence makes a Metron-linked character eligible for enrichment."`
	SeriesIncompleteFields    []string `json:"seriesIncompleteFields" doc:"Series fields whose absence makes a Metron-linked series eligible for enrichment."`
	ArcIncompleteFields       []string `json:"arcIncompleteFields" doc:"Story-arc fields whose absence makes a Metron-linked arc eligible for enrichment."`
	ResourceOrder             []string `json:"resourceOrder" doc:"Maintenance resource processing order. Contains comics, characters, series, and arcs exactly once."`
}

type MetronComicScanStatus struct {
	Settings        MetronComicScanSettings `json:"settings"`
	Running         bool                    `json:"running"`
	StartedAt       string                  `json:"startedAt,omitempty"`
	FinishedAt      string                  `json:"finishedAt,omitempty"`
	StopReason      string                  `json:"stopReason,omitempty"`
	Scanned         int                     `json:"scanned"`
	Updated         int                     `json:"updated"`
	Failed          int                     `json:"failed"`
	LastError       string                  `json:"lastError,omitempty"`
	CurrentResource string                  `json:"currentResource,omitempty"`
	CallsUsedToday  int                     `json:"callsUsedToday"`
	CallsLeftToday  int                     `json:"callsLeftToday"`
	UsageDate       string                  `json:"usageDate"`
}

type MetronComicScanEvent struct {
	Scan MetronComicScanStatus `json:"scan" doc:"Current Metron maintenance settings, quota, and progress."`
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
		CharacterIncompleteFields: append(
			[]string(nil),
			metronCharacterIncompleteFields...,
		),
		SeriesIncompleteFields: append([]string(nil), metronSeriesIncompleteFields...),
		ArcIncompleteFields:    append([]string(nil), metronArcIncompleteFields...),
		ResourceOrder:          append([]string(nil), metronMaintenanceResourceOrder...),
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
	var err error
	if settings.IncompleteFields, err = normalizeIncompleteFields(
		settings.IncompleteFields,
		metronComicIncompleteFields,
		metronComicIncompleteConditions,
		settings.ScanComics,
		"incompleteFields",
	); err != nil {
		return err
	}
	if settings.CharacterIncompleteFields, err = normalizeIncompleteFields(
		settings.CharacterIncompleteFields,
		metronCharacterIncompleteFields,
		metronCharacterIncompleteConditions,
		settings.ScanCharacters,
		"characterIncompleteFields",
	); err != nil {
		return err
	}
	if settings.SeriesIncompleteFields, err = normalizeIncompleteFields(
		settings.SeriesIncompleteFields,
		metronSeriesIncompleteFields,
		metronSeriesIncompleteConditions,
		settings.ScanSeries,
		"seriesIncompleteFields",
	); err != nil {
		return err
	}
	if settings.ArcIncompleteFields, err = normalizeIncompleteFields(
		settings.ArcIncompleteFields,
		metronArcIncompleteFields,
		metronArcIncompleteConditions,
		settings.ScanArcs,
		"arcIncompleteFields",
	); err != nil {
		return err
	}
	if settings.ResourceOrder, err = normalizeMetronMaintenanceResourceOrder(settings.ResourceOrder); err != nil {
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
	if !settings.ScanComics && !settings.ScanCharacters && !settings.ScanSeries && !settings.ScanArcs {
		return errors.New("at least one maintenance data type must be enabled")
	}
	return nil
}

func normalizeIncompleteFields(
	values []string,
	ordered []string,
	conditions map[string]string,
	required bool,
	fieldName string,
) ([]string, error) {
	selected := map[string]bool{}
	for _, field := range values {
		if _, ok := conditions[field]; !ok {
			return nil, fmt.Errorf("invalid %s field %q", fieldName, field)
		}
		selected[field] = true
	}
	if required && len(selected) == 0 {
		return nil, fmt.Errorf("%s must contain at least one field", fieldName)
	}
	normalized := make([]string, 0, len(selected))
	for _, field := range ordered {
		if selected[field] {
			normalized = append(normalized, field)
		}
	}
	return normalized, nil
}

func normalizeMetronMaintenanceResourceOrder(values []string) ([]string, error) {
	known := map[string]bool{}
	for _, resource := range metronMaintenanceResourceOrder {
		known[resource] = true
	}
	seen := map[string]bool{}
	normalized := make([]string, 0, len(metronMaintenanceResourceOrder))
	for _, resource := range values {
		resource = strings.ToLower(strings.TrimSpace(resource))
		if !known[resource] {
			return nil, fmt.Errorf("invalid maintenance resource %q", resource)
		}
		if !seen[resource] {
			seen[resource] = true
			normalized = append(normalized, resource)
		}
	}
	for _, resource := range metronMaintenanceResourceOrder {
		if !seen[resource] {
			normalized = append(normalized, resource)
		}
	}
	return normalized, nil
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
	if !settings.Enabled {
		return errors.New("metron maintenance is disabled")
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
	now := time.Now()
	var err error
	comics := []incompleteComicRow{}
	characters := []incompleteMetronRow{}
	series := []incompleteMetronRow{}
	arcs := []incompleteMetronRow{}

	if settings.ScanComics {
		comics, err = selectIncompleteComics(ctx, s.db, settings, now)
	}
	if err == nil && settings.ScanCharacters {
		characters, err = selectIncompleteCharacters(ctx, s.db, settings, now)
	}
	if err == nil && settings.ScanSeries {
		series, err = selectIncompleteSeries(ctx, s.db, settings, now)
	}
	if err == nil && settings.ScanArcs {
		arcs, err = selectIncompleteArcs(ctx, s.db, settings, now)
	}

	if err == nil {
		s.setScanned(len(comics) + len(characters) + len(series) + len(arcs))
		budget := metronMaintenanceBudget{
			db:       s.db,
			limit:    settings.DailyCallLimit,
			interval: time.Duration(settings.MinIntervalSeconds) * time.Second,
		}
		scanCtx := ctx
		needsUser := (settings.PullCharacterComics && len(characters) > 0) ||
			(settings.PullSeriesComics && len(series) > 0) ||
			(settings.PullArcComics && len(arcs) > 0)
		if needsUser {
			var userID int
			userID, err = ensureDefaultUser(ctx, s.db)
			if err == nil {
				scanCtx = context.WithValue(ctx, contextUserIDKey{}, userID)
			}
		}
		resourceOrder, orderErr := normalizeMetronMaintenanceResourceOrder(settings.ResourceOrder)
		if err == nil {
			err = orderErr
		}
		for _, resource := range resourceOrder {
			if err != nil {
				break
			}
			switch resource {
			case "comics":
				if len(comics) > 0 {
					s.setCurrentResource(resource)
					err = s.scanIncompleteComics(scanCtx, comics, &budget)
				}
			case "characters":
				if len(characters) > 0 {
					s.setCurrentResource(resource)
					err = s.scanIncompleteCharacters(scanCtx, characters, settings.PullCharacterComics, &budget)
				}
			case "series":
				if len(series) > 0 {
					s.setCurrentResource(resource)
					err = s.scanIncompleteSeries(scanCtx, series, settings.PullSeriesComics, &budget)
				}
			case "arcs":
				if len(arcs) > 0 {
					s.setCurrentResource(resource)
					err = s.scanIncompleteArcs(scanCtx, arcs, settings.PullArcComics, &budget)
				}
			}
		}
		if errors.Is(err, errMetronMaintenanceQuotaUsed) {
			s.setStopReason("daily quota used")
			err = nil
		}
	}
	s.mu.Lock()
	s.status.Running = false
	s.status.CurrentResource = ""
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

var errMetronMaintenanceQuotaUsed = errors.New("daily Metron maintenance quota used")

type metronMaintenanceBudget struct {
	db          *sqlx.DB
	limit       int
	interval    time.Duration
	nextRequest time.Time
}

func (b *metronMaintenanceBudget) claim(ctx context.Context) error {
	if err := waitForComicScanInterval(ctx, &b.nextRequest, b.interval); err != nil {
		return err
	}
	claimed, err := claimMetronComicScanCall(ctx, b.db, b.limit, time.Now())
	if err != nil {
		return err
	}
	if !claimed {
		return errMetronMaintenanceQuotaUsed
	}
	return nil
}

func (s *metronComicScanner) scanIncompleteComics(ctx context.Context, rows []incompleteComicRow, budget *metronMaintenanceBudget) error {
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := budget.claim(ctx); err != nil {
			return err
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
				if markErr := markIncompleteComicChecked(ctx, s.db, row.ID, time.Now()); markErr != nil {
					s.recordFailure("comic", row.ID, fmt.Errorf("mark unmatched comic as checked: %w", markErr))
				}
				continue
			} else if len(matches) > 1 {
				fetchErr = fmt.Errorf("Comic Vine ID %d returned %d Metron issues", row.ComicVineID.Int64, len(matches))
			} else {
				if err := budget.claim(ctx); err != nil {
					return err
				}
				issue, fetchErr = s.client.GetIssue(ctx, matches[0].ID)
			}
		}
		if fetchErr != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
			s.recordFailure("comic", row.ID, fmt.Errorf("fetch Metron issue: %w", fetchErr))
			continue
		}
		if comicVineIDsDiffer(row.ComicVineID, issue.ComicVineID) {
			log.Printf("Metron comic pull skipped: comic_id=%d local_comic_vine_id=%d metron_comic_vine_id=%d", row.ID, row.ComicVineID.Int64, issue.ComicVineID)
			if err := markIncompleteComicChecked(ctx, s.db, row.ID, time.Now()); err != nil {
				s.recordFailure("comic", row.ID, fmt.Errorf("mark Comic Vine mismatch as checked: %w", err))
			}
			continue
		}
		if enrichErr := enrichIncompleteComicFromMetron(ctx, s.db, s.covers, row.ID, *issue); enrichErr != nil {
			s.recordFailure("comic", row.ID, enrichErr)
		} else {
			s.incrementUpdated()
		}
	}
	return nil
}

func (s *metronComicScanner) scanIncompleteCharacters(ctx context.Context, rows []incompleteMetronRow, pullComics bool, budget *metronMaintenanceBudget) error {
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := budget.claim(ctx); err != nil {
			return err
		}
		character, fetchErr := s.client.GetCharacter(ctx, row.MetronID)
		if fetchErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			s.recordFailure("character", row.ID, fmt.Errorf("fetch Metron character: %w", fetchErr))
			continue
		}
		if enrichErr := enrichIncompleteCharacterFromMetron(ctx, s.db, s.covers, row.ID, *character); enrichErr != nil {
			s.recordFailure("character", row.ID, enrichErr)
			continue
		}
		if pullComics {
			if importErr := s.pullCharacterComicList(ctx, row, budget); importErr != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				if errors.Is(importErr, errMetronMaintenanceQuotaUsed) {
					return importErr
				}
				s.recordFailure("character", row.ID, fmt.Errorf("pull character comic list: %w", importErr))
				continue
			}
		}
		if markErr := markMetronSynced(ctx, s.db, metronResourceCharacter, row.MetronID, metron.FetchInfo{}); markErr != nil {
			s.recordFailure("character", row.ID, markErr)
			continue
		}
		s.incrementUpdated()
	}
	return nil
}

func (s *metronComicScanner) scanIncompleteSeries(ctx context.Context, rows []incompleteMetronRow, pullComics bool, budget *metronMaintenanceBudget) error {
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := budget.claim(ctx); err != nil {
			return err
		}
		metadata, fetchErr := s.client.GetSeries(ctx, row.MetronID)
		if fetchErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			s.recordFailure("series", row.ID, fmt.Errorf("fetch Metron series: %w", fetchErr))
			continue
		}
		if enrichErr := enrichIncompleteSeriesFromMetron(ctx, s.db, row.ID, *metadata); enrichErr != nil {
			s.recordFailure("series", row.ID, enrichErr)
			continue
		}
		if pullComics {
			if listErr := s.pullSeriesComicList(ctx, row, budget); listErr != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				if errors.Is(listErr, errMetronMaintenanceQuotaUsed) {
					return listErr
				}
				s.recordFailure("series", row.ID, fmt.Errorf("pull series comic list: %w", listErr))
				continue
			}
		}
		if markErr := markMetronSynced(ctx, s.db, metronResourceSeries, row.MetronID, metron.FetchInfo{}); markErr != nil {
			s.recordFailure("series", row.ID, markErr)
			continue
		}
		s.incrementUpdated()
	}
	return nil
}

func (s *metronComicScanner) scanIncompleteArcs(ctx context.Context, rows []incompleteMetronRow, pullComics bool, budget *metronMaintenanceBudget) error {
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := budget.claim(ctx); err != nil {
			return err
		}
		metadata, fetchErr := s.client.GetArcMetadata(ctx, row.MetronID)
		if fetchErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			s.recordFailure("arc", row.ID, fmt.Errorf("fetch Metron arc: %w", fetchErr))
			continue
		}
		if enrichErr := enrichIncompleteArcFromMetron(ctx, s.db, row.ID, *metadata); enrichErr != nil {
			s.recordFailure("arc", row.ID, enrichErr)
			continue
		}
		if pullComics {
			issues := []metron.Issue{}
			listErr := s.client.EachArcIssuePageWithRequest(
				ctx,
				row.MetronID,
				func() error { return budget.claim(ctx) },
				func(page []metron.Issue, _ int) error {
					issues = append(issues, page...)
					return nil
				},
			)
			if listErr != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				if errors.Is(listErr, errMetronMaintenanceQuotaUsed) {
					return listErr
				}
				s.recordFailure("arc", row.ID, fmt.Errorf("pull arc comic list: %w", listErr))
				continue
			}
			if importErr := importMetronArcWithOptions(
				ctx,
				s.db,
				s.client,
				s.covers,
				metron.MetronArc{ID: row.MetronID, Issues: issues},
				true,
				func(int, int, string) {},
				defaultMetronImportOptions(),
			); importErr != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
				s.recordFailure("arc", row.ID, fmt.Errorf("import arc comic list: %w", importErr))
				continue
			}
		}
		if markErr := markMetronSynced(ctx, s.db, metronResourceArc, row.MetronID, metron.FetchInfo{}); markErr != nil {
			s.recordFailure("arc", row.ID, markErr)
			continue
		}
		s.incrementUpdated()
	}
	return nil
}

func (s *metronComicScanner) pullCharacterComicList(ctx context.Context, row incompleteMetronRow, budget *metronMaintenanceBudget) error {
	options := defaultMetronImportOptions()
	return s.client.EachCharacterIssuePageWithRequest(
		ctx,
		row.MetronID,
		func() error { return budget.claim(ctx) },
		func(issues []metron.Issue, _ int) error {
			for _, issue := range issues {
				if err := ctx.Err(); err != nil {
					return err
				}
				comic, err := importMetronCharacterAppearanceIssueWithOptions(
					ctx,
					s.db,
					s.client,
					s.covers,
					issue,
					options,
				)
				if err != nil {
					return err
				}
				if err := linkCharacterAppearance(ctx, s.db, row.ID, comic.ID); err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func (s *metronComicScanner) pullSeriesComicList(ctx context.Context, row incompleteMetronRow, budget *metronMaintenanceBudget) error {
	options := defaultMetronImportOptions()
	return s.client.EachSeriesIssuePageWithRequest(
		ctx,
		row.MetronID,
		func() error { return budget.claim(ctx) },
		func(issues []metron.Issue, _ int) error {
			_, err := importMetronSeriesWithProgressOptions(
				ctx,
				s.db,
				s.client,
				s.covers,
				issues,
				func(int, int, string) {},
				options,
			)
			return err
		},
	)
}

type incompleteComicRow struct {
	ID          int           `db:"id"`
	MetronID    sql.NullInt64 `db:"metron_issue_id"`
	ComicVineID sql.NullInt64 `db:"comic_vine_id"`
}

type incompleteMetronRow struct {
	ID       int `db:"id"`
	MetronID int `db:"metron_id"`
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

func selectIncompleteCharacters(ctx context.Context, db *sqlx.DB, settings MetronComicScanSettings, now time.Time) ([]incompleteMetronRow, error) {
	return selectIncompleteMetronRows(
		ctx,
		db,
		"characters",
		"ch",
		"metron_character_id",
		metronResourceCharacter,
		settings.CharacterIncompleteFields,
		metronCharacterIncompleteConditions,
		settings.RecheckCooldownDays,
		now,
	)
}

func selectIncompleteSeries(ctx context.Context, db *sqlx.DB, settings MetronComicScanSettings, now time.Time) ([]incompleteMetronRow, error) {
	return selectIncompleteMetronRows(
		ctx,
		db,
		"series",
		"s",
		"metron_series_id",
		metronResourceSeries,
		settings.SeriesIncompleteFields,
		metronSeriesIncompleteConditions,
		settings.RecheckCooldownDays,
		now,
	)
}

func selectIncompleteArcs(ctx context.Context, db *sqlx.DB, settings MetronComicScanSettings, now time.Time) ([]incompleteMetronRow, error) {
	return selectIncompleteMetronRows(
		ctx,
		db,
		"arcs",
		"a",
		"metron_arc_id",
		metronResourceArc,
		settings.ArcIncompleteFields,
		metronArcIncompleteConditions,
		settings.RecheckCooldownDays,
		now,
	)
}

func selectIncompleteMetronRows(
	ctx context.Context,
	db *sqlx.DB,
	table string,
	alias string,
	metronIDColumn string,
	resourceType string,
	fields []string,
	availableConditions map[string]string,
	cooldownDays int,
	now time.Time,
) ([]incompleteMetronRow, error) {
	conditions := make([]string, 0, len(fields))
	for _, field := range fields {
		if condition, ok := availableConditions[field]; ok {
			conditions = append(conditions, condition)
		}
	}
	if len(conditions) == 0 {
		return []incompleteMetronRow{}, nil
	}

	query := fmt.Sprintf(`
		SELECT %[1]s.id, %[1]s.%[2]s AS metron_id
		FROM %[3]s %[1]s
		LEFT JOIN metron_sync_states ms
			ON ms.resource_type = ? AND ms.metron_id = %[1]s.%[2]s
		WHERE %[1]s.%[2]s IS NOT NULL
			AND (%[4]s)
	`, alias, metronIDColumn, table, strings.Join(conditions, " OR "))
	args := []any{resourceType}
	if cooldownDays > 0 {
		cutoff := now.Add(-time.Duration(cooldownDays) * 24 * time.Hour).UTC().Format(time.RFC3339)
		query += ` AND (COALESCE(ms.synced_at, '') = '' OR ms.synced_at <= ?)`
		args = append(args, cutoff)
	}
	query += fmt.Sprintf(` ORDER BY %s.id`, alias)

	rows := []incompleteMetronRow{}
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

func enrichIncompleteCharacterFromMetron(ctx context.Context, db *sqlx.DB, covers *CoverCache, characterID int, character metron.MetronCharacter) error {
	image := ""
	if strings.TrimSpace(character.Image) != "" {
		var current string
		if err := db.GetContext(ctx, &current, `SELECT image FROM characters WHERE id = ?`, characterID); err != nil {
			return fmt.Errorf("read character image: %w", err)
		}
		if strings.TrimSpace(current) == "" {
			var err error
			image, err = localCoverURL(ctx, covers, character.Image)
			if err != nil {
				return fmt.Errorf("cache character image: %w", err)
			}
		}
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start character metadata update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE characters
		SET description = CASE WHEN TRIM(description) = '' THEN ? ELSE description END,
			image = CASE WHEN TRIM(image) = '' THEN ? ELSE image END
		WHERE id = ?
	`, character.Description, image, characterID); err != nil {
		return fmt.Errorf("update character metadata: %w", err)
	}
	for _, alias := range cleanAliases(character.Aliases) {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO character_aliases (character_id, alias)
			VALUES (?, ?)
		`, characterID, alias); err != nil {
			return fmt.Errorf("update character aliases: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save character metadata: %w", err)
	}
	return nil
}

func enrichIncompleteSeriesFromMetron(ctx context.Context, db *sqlx.DB, seriesID int, series metron.Series) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("start series metadata update: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `
		UPDATE series
		SET publisher = CASE WHEN TRIM(publisher) = '' THEN ? ELSE publisher END,
			series_year = CASE WHEN series_year = 0 AND ? > 0 THEN ? ELSE series_year END,
			volume = CASE WHEN volume = 0 AND ? > 0 THEN ? ELSE volume END,
			year_end = CASE WHEN year_end = 0 AND ? > 0 THEN ? ELSE year_end END,
			issue_count = CASE WHEN issue_count = 0 AND ? > 0 THEN ? ELSE issue_count END,
			description = CASE WHEN TRIM(description) = '' THEN ? ELSE description END
		WHERE id = ?
	`, series.Publisher,
		series.YearBegan, series.YearBegan,
		series.Volume, series.Volume,
		series.YearEnd, series.YearEnd,
		series.IssueCount, series.IssueCount,
		series.Description,
		seriesID,
	); err != nil {
		return fmt.Errorf("update series metadata: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE comics
		SET series = (SELECT name FROM series WHERE id = ?),
			series_year = (SELECT series_year FROM series WHERE id = ?)
		WHERE series_id = ?
	`, seriesID, seriesID, seriesID); err != nil {
		return fmt.Errorf("update series comics: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("save series metadata: %w", err)
	}
	return nil
}

func enrichIncompleteArcFromMetron(ctx context.Context, db *sqlx.DB, arcID int, arc metron.MetronArc) error {
	if _, err := db.ExecContext(ctx, `
		UPDATE arcs
		SET description = CASE WHEN TRIM(description) = '' THEN ? ELSE description END,
			image = CASE WHEN TRIM(image) = '' THEN ? ELSE image END
		WHERE id = ?
	`, arc.Description, arc.Image, arcID); err != nil {
		return fmt.Errorf("update arc metadata: %w", err)
	}
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
				return fmt.Errorf("cache cover: %w", err)
			}
		} else {
			cover = current
		}
	}
	_, err := db.ExecContext(ctx, `
		UPDATE comics SET
			metron_issue_id = COALESCE(metron_issue_id, (
				SELECT ? WHERE NOT EXISTS (
					SELECT 1 FROM comics linked
					WHERE linked.metron_issue_id = ? AND linked.id <> ?
				)
			)),
			comic_vine_id = COALESCE(comic_vine_id, (
				SELECT ? WHERE NOT EXISTS (
					SELECT 1 FROM comics linked
					WHERE linked.comic_vine_id = ? AND linked.id <> ?
				)
			)),
			publisher = CASE WHEN TRIM(publisher) = '' THEN ? ELSE publisher END,
			cover_date = CASE WHEN TRIM(cover_date) = '' THEN ? ELSE cover_date END,
			cover_image = CASE WHEN TRIM(cover_image) = '' THEN ? ELSE cover_image END,
			description = CASE WHEN TRIM(description) = '' THEN ? ELSE description END
		WHERE id = ?
	`, nullablePositiveID(issue.ID), issue.ID, comicID,
		nullablePositiveID(issue.ComicVineID), issue.ComicVineID, comicID,
		issue.Publisher, issue.CoverDate, cover, issue.Description, comicID)
	if err != nil {
		return fmt.Errorf("update comic metadata: %w", err)
	}
	// The issue response already contains lightweight arc and character data. A
	// nil client prevents these helpers from making any detail requests.
	options := MetronImportOptions{Mode: "basic"}
	if err := syncMetronIssueArcsWithOptions(ctx, db, nil, comicID, issue, options); err != nil {
		return fmt.Errorf("sync comic arcs: %w", err)
	}
	if err := syncMetronIssueCharactersWithOptions(ctx, db, nil, covers, comicID, issue, options); err != nil {
		return fmt.Errorf("sync comic characters: %w", err)
	}
	if err := markIncompleteComicChecked(ctx, db, comicID, time.Now()); err != nil {
		return fmt.Errorf("mark comic as synced: %w", err)
	}
	return nil
}

func comicVineIDsDiffer(local sql.NullInt64, metronID int) bool {
	return local.Valid && local.Int64 > 0 && metronID > 0 && local.Int64 != int64(metronID)
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
	defer func() { _ = tx.Rollback() }()
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
func (s *metronComicScanner) setCurrentResource(resource string) {
	s.mu.Lock()
	s.status.CurrentResource = resource
	s.mu.Unlock()
	s.broadcastSnapshot()
}
func (s *metronComicScanner) recordFailure(resourceType string, localID int, err error) {
	message := fmt.Sprintf("%s %d: %v", resourceType, localID, err)
	log.Printf("Metron maintenance failed: %s", message)
	s.mu.Lock()
	s.status.Failed++
	s.status.LastError = message
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
	huma.Register(api, huma.Operation{OperationID: "getMetronComicScan", Tags: []string{tagMetron}, Summary: "Get Metron maintenance settings and status", Method: http.MethodGet, Path: "/metron/scans/comics", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		return &MetronComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
	sse.Register(api, huma.Operation{OperationID: "streamMetronComicScan", Tags: []string{tagMetron}, Summary: "Stream Metron maintenance status", Description: "Streams an initial snapshot and live Metron maintenance settings, quota, and progress updates.", Method: http.MethodGet, Path: "/metron/scans/comics/events", Errors: []int{401, 403, 500}}, map[string]any{"scan": MetronComicScanEvent{}}, func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return
		}
		streamMetronComicScan(ctx, scanner, func(event MetronComicScanEvent) error { return send.Data(event) })
	})
	huma.Register(api, huma.Operation{OperationID: "updateMetronComicScan", Tags: []string{tagMetron}, Summary: "Update Metron maintenance settings", Method: http.MethodPut, Path: "/metron/scans/comics", Errors: []int{400, 401, 403, 500}}, func(ctx context.Context, input *UpdateMetronComicScanSettingsInput) (*MetronComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := validateMetronComicScanSettings(&input.Body); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := saveMetronComicScanSettings(ctx, db, input.Body); err != nil {
			return nil, huma.Error500InternalServerError("failed to save Metron maintenance settings")
		}
		select {
		case scanner.wake <- struct{}{}:
		default:
		}
		scanner.broadcastSnapshot()
		return &MetronComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "triggerMetronComicScan", Tags: []string{tagMetron}, Summary: "Trigger Metron maintenance", Method: http.MethodPost, Path: "/metron/scans/comics/trigger", DefaultStatus: http.StatusAccepted, Errors: []int{400, 401, 403, 409, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicScanStatusOutput, error) {
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
	huma.Register(api, huma.Operation{OperationID: "stopMetronComicScan", Tags: []string{tagMetron}, Summary: "Stop Metron maintenance", Method: http.MethodPost, Path: "/metron/scans/comics/stop", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicScanStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		scanner.stopScan("stopped by admin")
		return &MetronComicScanStatusOutput{Body: scanner.snapshot(ctx)}, nil
	})
}
