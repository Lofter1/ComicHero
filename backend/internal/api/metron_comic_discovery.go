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
	metronComicDiscoverySettingsKey = "metron_comic_discovery_settings"
	metronComicDiscoveryLastRunKey  = "metron_comic_discovery_last_scheduled_date"
)

type MetronComicDiscoverySettings struct {
	Enabled          bool     `json:"enabled"`
	PullComics       bool     `json:"pullComics"`
	PullReadingLists bool     `json:"pullReadingLists"`
	Schedule         string   `json:"schedule" enum:"daily,weekly,monthly"`
	Weekdays         []string `json:"weekdays,omitempty"`
	MonthDay         int      `json:"monthDay,omitempty" minimum:"1" maximum:"31"`
	StartTime        string   `json:"startTime" example:"03:00"`
	PublisherName    string   `json:"publisherName,omitempty"`
	SeriesName       string   `json:"seriesName,omitempty"`
}

type MetronComicDiscoveryStatus struct {
	Settings       MetronComicDiscoverySettings `json:"settings"`
	Running        bool                         `json:"running"`
	StartedAt      string                       `json:"startedAt,omitempty"`
	FinishedAt     string                       `json:"finishedAt,omitempty"`
	ModifiedAfter  string                       `json:"modifiedAfter,omitempty"`
	StopReason     string                       `json:"stopReason,omitempty"`
	Found          int                          `json:"found"`
	Imported       int                          `json:"imported"`
	AlreadyPresent int                          `json:"alreadyPresent"`
	Failed         int                          `json:"failed"`
}

type MetronComicDiscoveryEvent struct {
	Discovery MetronComicDiscoveryStatus `json:"discovery"`
}

type metronComicDiscovery struct {
	db               *sqlx.DB
	client           *metron.Client
	covers           *CoverCache
	mu               sync.Mutex
	status           MetronComicDiscoveryStatus
	cancel           context.CancelFunc
	wake             chan struct{}
	shutdown         context.CancelFunc
	nextSubscriberID uint64
	subscribers      map[uint64]chan MetronComicDiscoveryStatus
}

func NewMetronComicDiscovery(db *sqlx.DB, client *metron.Client, covers *CoverCache) *metronComicDiscovery {
	return &metronComicDiscovery{db: db, client: client, covers: covers, wake: make(chan struct{}, 1), subscribers: map[uint64]chan MetronComicDiscoveryStatus{}}
}

func (d *metronComicDiscovery) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	d.shutdown = cancel
	go d.scheduleLoop(ctx)
}

func (d *metronComicDiscovery) Stop() {
	if d.shutdown != nil {
		d.shutdown()
	}
	d.stop("server stopped")
}

func defaultMetronComicDiscoverySettings() MetronComicDiscoverySettings {
	return MetronComicDiscoverySettings{PullComics: true, Schedule: "daily", Weekdays: []string{"monday"}, MonthDay: 1, StartTime: "03:00"}
}

func loadMetronComicDiscoverySettings(ctx context.Context, db *sqlx.DB) (MetronComicDiscoverySettings, error) {
	settings := defaultMetronComicDiscoverySettings()
	var value string
	if err := db.GetContext(ctx, &value, `SELECT value FROM app_settings WHERE key = ?`, metronComicDiscoverySettingsKey); err != nil {
		if err == sql.ErrNoRows {
			return settings, nil
		}
		return settings, err
	}
	var stored map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &stored); err != nil {
		return settings, err
	}
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return settings, err
	}
	if _, hasComics := stored["pullComics"]; !hasComics {
		if _, hasLists := stored["pullReadingLists"]; !hasLists {
			settings.PullComics = true
		}
	}
	return settings, nil
}

func validateMetronComicDiscoverySettings(settings *MetronComicDiscoverySettings) error {
	settings.Schedule = strings.ToLower(strings.TrimSpace(settings.Schedule))
	settings.PublisherName = strings.TrimSpace(settings.PublisherName)
	settings.SeriesName = strings.TrimSpace(settings.SeriesName)
	if !settings.PullComics && !settings.PullReadingLists {
		return errors.New("at least one content type must be selected")
	}
	if settings.Schedule != "daily" && settings.Schedule != "weekly" && settings.Schedule != "monthly" {
		return errors.New("schedule must be daily, weekly, or monthly")
	}
	if _, err := time.Parse("15:04", settings.StartTime); err != nil {
		return errors.New("startTime must use HH:MM")
	}
	seen := map[string]bool{}
	days := make([]string, 0, len(settings.Weekdays))
	for _, day := range settings.Weekdays {
		day = strings.ToLower(strings.TrimSpace(day))
		if _, ok := weekdayNames[day]; !ok {
			return fmt.Errorf("invalid weekday %q", day)
		}
		if !seen[day] {
			seen[day] = true
			days = append(days, day)
		}
	}
	sort.Strings(days)
	settings.Weekdays = days
	if settings.Schedule == "weekly" && len(days) == 0 {
		return errors.New("weekly schedules need at least one weekday")
	}
	if settings.MonthDay < 1 || settings.MonthDay > 31 {
		return errors.New("monthDay must be between 1 and 31")
	}
	return nil
}

func saveMetronComicDiscoverySettings(ctx context.Context, db *sqlx.DB, settings MetronComicDiscoverySettings) error {
	value, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, metronComicDiscoverySettingsKey, string(value))
	return err
}

func (d *metronComicDiscovery) scheduleLoop(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			d.checkSchedule(ctx, time.Now())
		case <-d.wake:
			d.checkSchedule(ctx, time.Now())
		}
	}
}

func (d *metronComicDiscovery) checkSchedule(ctx context.Context, now time.Time) {
	settings, err := loadMetronComicDiscoverySettings(ctx, d.db)
	if err != nil || !settings.Enabled || now.Format("15:04") != settings.StartTime || !discoveryScheduleMatches(settings, now) {
		return
	}
	date := now.Format("2006-01-02")
	var last string
	_ = d.db.GetContext(ctx, &last, `SELECT value FROM app_settings WHERE key = ?`, metronComicDiscoveryLastRunKey)
	if last == date || d.trigger("scheduled", now) != nil {
		return
	}
	_, _ = d.db.ExecContext(ctx, `INSERT INTO app_settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, metronComicDiscoveryLastRunKey, date)
}

func discoveryScheduleMatches(settings MetronComicDiscoverySettings, now time.Time) bool {
	switch settings.Schedule {
	case "weekly":
		for _, day := range settings.Weekdays {
			if weekdayNames[day] == now.Weekday() {
				return true
			}
		}
		return false
	case "monthly":
		return now.Day() == min(settings.MonthDay, daysInMonth(now))
	default:
		return true
	}
}

func daysInMonth(value time.Time) int {
	return time.Date(value.Year(), value.Month()+1, 0, 0, 0, 0, 0, value.Location()).Day()
}

func discoveryModifiedAfter(schedule string, now time.Time) time.Time {
	switch schedule {
	case "weekly":
		return now.AddDate(0, 0, -7)
	case "monthly":
		previousMonth := now.AddDate(0, -1, -now.Day()+1)
		day := min(now.Day(), daysInMonth(previousMonth))
		return time.Date(previousMonth.Year(), previousMonth.Month(), day, now.Hour(), now.Minute(), now.Second(), now.Nanosecond(), now.Location())
	default:
		return now.AddDate(0, 0, -1)
	}
}

func (d *metronComicDiscovery) trigger(reason string, now time.Time) error {
	settings, err := loadMetronComicDiscoverySettings(context.Background(), d.db)
	if err != nil {
		return err
	}
	if !settings.Enabled {
		return errors.New("automatic comic discovery is disabled")
	}
	d.mu.Lock()
	if d.status.Running {
		d.mu.Unlock()
		return errors.New("comic discovery is already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	modifiedAfter := discoveryModifiedAfter(settings.Schedule, now).UTC().Format(time.RFC3339)
	d.cancel = cancel
	d.status = MetronComicDiscoveryStatus{Settings: settings, Running: true, StartedAt: time.Now().UTC().Format(time.RFC3339), ModifiedAfter: modifiedAfter, StopReason: reason}
	d.mu.Unlock()
	d.broadcast()
	go d.run(ctx, settings, modifiedAfter)
	return nil
}

func (d *metronComicDiscovery) stop(reason string) bool {
	d.mu.Lock()
	if !d.status.Running || d.cancel == nil {
		d.mu.Unlock()
		return false
	}
	d.status.StopReason = reason
	d.cancel()
	d.mu.Unlock()
	d.broadcast()
	return true
}

func (d *metronComicDiscovery) run(ctx context.Context, settings MetronComicDiscoverySettings, modifiedAfter string) {
	var runErr error
	if settings.PullComics {
		issues, err := d.client.SearchModifiedIssues(ctx, metron.IssueModifiedSearchOptions{ModifiedAfter: modifiedAfter, PublisherName: settings.PublisherName, SeriesName: settings.SeriesName})
		if err != nil {
			runErr = err
		} else {
			d.update(func(status *MetronComicDiscoveryStatus) { status.Found = len(issues) })
			for _, issue := range issues {
				if ctx.Err() != nil {
					break
				}
				if _, exists, checkErr := existingComicIDByMetronIssueID(ctx, d.db, issue.ID); checkErr != nil {
					d.increment("failed")
					continue
				} else if exists {
					d.increment("present")
					continue
				}
				if _, importErr := importMetronComicWithOptions(ctx, d.db, d.client, d.covers, issue, MetronImportOptions{Mode: "basic"}); importErr != nil {
					d.increment("failed")
				} else {
					d.increment("imported")
				}
			}
		}
	}
	if settings.PullReadingLists && ctx.Err() == nil {
		lists, err := d.client.SearchModifiedReadingLists(ctx, modifiedAfter)
		if err != nil {
			if runErr == nil {
				runErr = err
			}
		} else {
			d.update(func(status *MetronComicDiscoveryStatus) { status.Found += len(lists) })
			defaultUserID, userErr := ensureDefaultUser(ctx, d.db)
			if userErr != nil {
				runErr = userErr
			} else {
				userCtx := context.WithValue(ctx, contextUserIDKey{}, defaultUserID)
				for _, list := range lists {
					if ctx.Err() != nil {
						break
					}
					if _, exists, checkErr := existingReadingOrderIDByMetronID(ctx, d.db, list.ID); checkErr != nil {
						d.increment("failed")
						continue
					} else if exists {
						d.increment("present")
						continue
					}
					detail, detailErr := d.client.GetReadingList(ctx, list.ID)
					if detailErr != nil {
						d.increment("failed")
						continue
					}
					if importErr := importMetronReadingListWithOptions(userCtx, d.db, d.client, d.covers, *detail, false, func(int, int, string) {}, defaultMetronImportOptions()); importErr != nil {
						d.increment("failed")
					} else {
						d.increment("imported")
					}
				}
			}
		}
	}
	d.mu.Lock()
	d.status.Running = false
	d.status.FinishedAt = time.Now().UTC().Format(time.RFC3339)
	if runErr != nil {
		d.status.StopReason = runErr.Error()
	} else if ctx.Err() != nil && d.status.StopReason == "" {
		d.status.StopReason = "stopped"
	} else if d.status.StopReason == "manual" || d.status.StopReason == "scheduled" {
		d.status.StopReason = "complete"
	}
	d.cancel = nil
	d.mu.Unlock()
	d.broadcast()
}

func (d *metronComicDiscovery) increment(kind string) {
	d.update(func(status *MetronComicDiscoveryStatus) {
		switch kind {
		case "imported":
			status.Imported++
		case "present":
			status.AlreadyPresent++
		default:
			status.Failed++
		}
	})
}
func (d *metronComicDiscovery) update(change func(*MetronComicDiscoveryStatus)) {
	d.mu.Lock()
	change(&d.status)
	d.mu.Unlock()
	d.broadcast()
}

func (d *metronComicDiscovery) snapshot(ctx context.Context) MetronComicDiscoveryStatus {
	settings, _ := loadMetronComicDiscoverySettings(ctx, d.db)
	d.mu.Lock()
	status := d.status
	d.mu.Unlock()
	status.Settings = settings
	return status
}

func (d *metronComicDiscovery) subscribe(ctx context.Context) (<-chan MetronComicDiscoveryStatus, func()) {
	d.mu.Lock()
	d.nextSubscriberID++
	id := d.nextSubscriberID
	ch := make(chan MetronComicDiscoveryStatus, 16)
	d.subscribers[id] = ch
	d.mu.Unlock()
	ch <- d.snapshot(ctx)
	return ch, func() {
		d.mu.Lock()
		if current, ok := d.subscribers[id]; ok {
			delete(d.subscribers, id)
			close(current)
		}
		d.mu.Unlock()
	}
}

func (d *metronComicDiscovery) broadcast() {
	status := d.snapshot(context.Background())
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, ch := range d.subscribers {
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

func streamMetronComicDiscovery(ctx context.Context, discovery *metronComicDiscovery, send func(MetronComicDiscoveryEvent) error) {
	updates, unsubscribe := discovery.subscribe(ctx)
	defer unsubscribe()
	for {
		select {
		case <-ctx.Done():
			return
		case status, ok := <-updates:
			if !ok || send(MetronComicDiscoveryEvent{Discovery: status}) != nil {
				return
			}
		}
	}
}

type MetronComicDiscoveryStatusOutput struct{ Body MetronComicDiscoveryStatus }
type UpdateMetronComicDiscoverySettingsInput struct{ Body MetronComicDiscoverySettings }

func RegisterMetronComicDiscoveryRoutes(api huma.API, db *sqlx.DB, discovery *metronComicDiscovery) {
	huma.Register(api, huma.Operation{OperationID: "getMetronComicDiscovery", Tags: []string{tagMetron}, Summary: "Get automatic comic discovery", Method: http.MethodGet, Path: "/metron/discovery/comics", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicDiscoveryStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		return &MetronComicDiscoveryStatusOutput{Body: discovery.snapshot(ctx)}, nil
	})
	sse.Register(api, huma.Operation{OperationID: "streamMetronComicDiscovery", Tags: []string{tagMetron}, Summary: "Stream automatic comic discovery", Method: http.MethodGet, Path: "/metron/discovery/comics/events", Errors: []int{401, 403, 500}}, map[string]any{"discovery": MetronComicDiscoveryEvent{}}, func(ctx context.Context, _ *struct{}, send sse.Sender) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return
		}
		streamMetronComicDiscovery(ctx, discovery, func(event MetronComicDiscoveryEvent) error { return send.Data(event) })
	})
	huma.Register(api, huma.Operation{OperationID: "updateMetronComicDiscovery", Tags: []string{tagMetron}, Summary: "Update automatic comic discovery", Method: http.MethodPut, Path: "/metron/discovery/comics", Errors: []int{400, 401, 403, 500}}, func(ctx context.Context, input *UpdateMetronComicDiscoverySettingsInput) (*MetronComicDiscoveryStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := validateMetronComicDiscoverySettings(&input.Body); err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		if err := saveMetronComicDiscoverySettings(ctx, db, input.Body); err != nil {
			return nil, huma.Error500InternalServerError("failed to save comic discovery settings")
		}
		select {
		case discovery.wake <- struct{}{}:
		default:
		}
		discovery.broadcast()
		return &MetronComicDiscoveryStatusOutput{Body: discovery.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "triggerMetronComicDiscovery", Tags: []string{tagMetron}, Summary: "Trigger automatic comic discovery", Method: http.MethodPost, Path: "/metron/discovery/comics/trigger", DefaultStatus: http.StatusAccepted, Errors: []int{400, 401, 403, 409, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicDiscoveryStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		if err := discovery.trigger("manual", time.Now()); err != nil {
			if strings.Contains(err.Error(), "already running") {
				return nil, huma.Error409Conflict(err.Error())
			}
			return nil, huma.Error400BadRequest(err.Error())
		}
		return &MetronComicDiscoveryStatusOutput{Body: discovery.snapshot(ctx)}, nil
	})
	huma.Register(api, huma.Operation{OperationID: "stopMetronComicDiscovery", Tags: []string{tagMetron}, Summary: "Stop automatic comic discovery", Method: http.MethodPost, Path: "/metron/discovery/comics/stop", Errors: []int{401, 403, 500}}, func(ctx context.Context, _ *struct{}) (*MetronComicDiscoveryStatusOutput, error) {
		if _, err := requireAdminUser(ctx, db); err != nil {
			return nil, err
		}
		discovery.stop("stopped by admin")
		return &MetronComicDiscoveryStatusOutput{Body: discovery.snapshot(ctx)}, nil
	})
}
