package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func newMetronComicScannerTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMetronComicScanSettingsRoundTrip(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	settings := MetronComicScanSettings{
		Enabled: true, ScanComics: true, ScanCharacters: true, ScanSeries: true, ScanArcs: true,
		PullCharacterComics: true, PullSeriesComics: true, PullArcComics: true,
		Schedule: "weekly", Weekdays: []string{"friday", "monday"}, StartTime: "03:15",
		DailyCallLimit: 12, MinIntervalSeconds: 20,
		IncompleteFields:          []string{"publisher", "comicVineId"},
		CharacterIncompleteFields: []string{"description"},
		SeriesIncompleteFields:    []string{"publisher"},
		ArcIncompleteFields:       []string{"image"},
		ResourceOrder:             []string{"arcs", "characters", "series", "comics"},
	}
	if err := validateMetronComicScanSettings(&settings); err != nil {
		t.Fatal(err)
	}
	if err := saveMetronComicScanSettings(context.Background(), db, settings); err != nil {
		t.Fatal(err)
	}
	got, err := loadMetronComicScanSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || !got.ScanCharacters || !got.ScanSeries || !got.ScanArcs ||
		!got.PullCharacterComics || !got.PullSeriesComics || !got.PullArcComics ||
		got.DailyCallLimit != 12 || got.MinIntervalSeconds != 20 ||
		len(got.Weekdays) != 2 || len(got.IncompleteFields) != 2 ||
		len(got.ResourceOrder) != 4 || got.ResourceOrder[0] != "arcs" {
		t.Fatalf("unexpected settings: %+v", got)
	}
}

func TestMetronComicScanLegacySettingsKeepDefaultIncompleteFields(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	if _, err := db.Exec(`INSERT INTO app_settings (key, value) VALUES (?, ?)`, metronComicScanSettingsKey, `{"enabled":true}`); err != nil {
		t.Fatal(err)
	}
	settings, err := loadMetronComicScanSettings(context.Background(), db)
	if err != nil {
		t.Fatal(err)
	}
	if len(settings.IncompleteFields) != len(metronComicIncompleteFields) {
		t.Fatalf("legacy incomplete fields = %v; want defaults %v", settings.IncompleteFields, metronComicIncompleteFields)
	}
	if len(settings.CharacterIncompleteFields) != len(metronCharacterIncompleteFields) ||
		len(settings.SeriesIncompleteFields) != len(metronSeriesIncompleteFields) ||
		len(settings.ArcIncompleteFields) != len(metronArcIncompleteFields) {
		t.Fatalf("legacy resource defaults were not preserved: %+v", settings)
	}
	if strings.Join(settings.ResourceOrder, ",") != strings.Join(metronMaintenanceResourceOrder, ",") {
		t.Fatalf("legacy resource order = %v; want %v", settings.ResourceOrder, metronMaintenanceResourceOrder)
	}
}

func TestMetronComicScanSettingsRequireKnownIncompleteFields(t *testing.T) {
	settings := defaultMetronComicScanSettings()
	settings.IncompleteFields = nil
	if err := validateMetronComicScanSettings(&settings); err == nil {
		t.Fatal("empty incomplete fields returned nil error")
	}

	settings = defaultMetronComicScanSettings()
	settings.IncompleteFields = append(settings.IncompleteFields, "unknown")
	if err := validateMetronComicScanSettings(&settings); err == nil {
		t.Fatal("unknown incomplete field returned nil error")
	}
}

func TestMetronMaintenanceCanScanLinkedResourcesWithoutComics(t *testing.T) {
	settings := defaultMetronComicScanSettings()
	settings.ScanComics = false
	settings.ScanCharacters = true
	settings.CharacterIncompleteFields = []string{"description", "aliases"}
	if err := validateMetronComicScanSettings(&settings); err != nil {
		t.Fatalf("validate character-only maintenance: %v", err)
	}

	settings.CharacterIncompleteFields = nil
	if err := validateMetronComicScanSettings(&settings); err == nil {
		t.Fatal("enabled character maintenance accepted no incomplete fields")
	}
}

func TestMetronMaintenanceResourceOrderIsNormalized(t *testing.T) {
	order, err := normalizeMetronMaintenanceResourceOrder([]string{" ARCS ", "comics", "arcs"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"arcs", "comics", "characters", "series"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v; want %v", order, want)
	}
	if _, err := normalizeMetronMaintenanceResourceOrder([]string{"publishers"}); err == nil {
		t.Fatal("unknown maintenance resource was accepted")
	}
}

func TestMetronComicScanDailyQuotaIsSharedAndResets(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	ctx := context.Background()
	dayOne := time.Date(2026, 7, 10, 10, 0, 0, 0, time.Local)
	for i := range 2 {
		claimed, err := claimMetronComicScanCall(ctx, db, 2, dayOne)
		if err != nil || !claimed {
			t.Fatalf("claim %d: claimed=%v err=%v", i, claimed, err)
		}
	}
	claimed, err := claimMetronComicScanCall(ctx, db, 2, dayOne)
	if err != nil || claimed {
		t.Fatalf("quota should be exhausted: claimed=%v err=%v", claimed, err)
	}
	claimed, err = claimMetronComicScanCall(ctx, db, 2, dayOne.Add(24*time.Hour))
	if err != nil || !claimed {
		t.Fatalf("quota should reset: claimed=%v err=%v", claimed, err)
	}
}

func TestMetronComicScanSubscriptionSendsSnapshotAndProgress(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	scanner := NewMetronComicScanner(db, nil, nil)
	updates, unsubscribe := scanner.subscribe(context.Background())
	defer unsubscribe()

	initial := <-updates
	if initial.Scanned != 0 {
		t.Fatalf("initial scanned = %d", initial.Scanned)
	}
	scanner.setScanned(3)
	progress := <-updates
	if progress.Scanned != 3 {
		t.Fatalf("progress scanned = %d", progress.Scanned)
	}
}

func TestMetronComicScanCooldownExcludesRecentlySyncedComics(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	if _, err := db.Exec(`CREATE TABLE comics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		series TEXT NOT NULL DEFAULT '',
		series_year INTEGER NOT NULL DEFAULT 0,
		issue TEXT NOT NULL DEFAULT '',
		publisher TEXT NOT NULL DEFAULT '',
		cover_date TEXT NOT NULL DEFAULT '',
		cover_image TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		read INTEGER NOT NULL DEFAULT 0,
		metron_issue_id INTEGER,
		comic_vine_id INTEGER,
		metron_synced_at TEXT NOT NULL DEFAULT ''
	)`); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	recentlySynced := now.Add(-1 * time.Hour).Format(time.RFC3339)
	longAgoSynced := now.Add(-40 * 24 * time.Hour).Format(time.RFC3339)

	// Row A: publisher filled, description permanently blank (Metron has none),
	// synced an hour ago -> still "incomplete" but should be skipped by the cooldown.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('A', 'Marvel', '1964-01-01', '/covers/a.jpg', '', 1, 101, ?)`, recentlySynced); err != nil {
		t.Fatal(err)
	}
	// Row B: never synced -> should always be selected.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('B', '', '', '', '', 2, NULL, '')`); err != nil {
		t.Fatal(err)
	}
	// Row C: synced 40 days ago, past the 30-day cooldown -> should be selected again.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('C', 'DC', '1980-01-01', '/covers/c.jpg', '', 3, 103, ?)`, longAgoSynced); err != nil {
		t.Fatal(err)
	}
	// Row D: all legacy metadata is complete, but the Comic Vine ID is missing.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('D', 'Image', '2020-01-01', '/covers/d.jpg', 'Complete', 4, NULL, '')`); err != nil {
		t.Fatal(err)
	}
	// Row E: no Metron ID yet, but its Comic Vine ID can be used to find one.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('E', '', '', '', '', NULL, 105, '')`); err != nil {
		t.Fatal(err)
	}
	// Row F: a recent unsuccessful Comic Vine lookup should respect the cooldown.
	if _, err := db.Exec(`INSERT INTO comics (series, publisher, cover_date, cover_image, description, metron_issue_id, comic_vine_id, metron_synced_at)
		VALUES ('F', '', '', '', '', NULL, 106, ?)`, recentlySynced); err != nil {
		t.Fatal(err)
	}

	settings := defaultMetronComicScanSettings()
	rows, err := selectIncompleteComics(context.Background(), db, settings, now)
	if err != nil {
		t.Fatal(err)
	}

	got := map[int]bool{}
	for _, r := range rows {
		got[r.ID] = true
	}
	if got[1] {
		t.Fatal("recently-synced comic with a permanently-blank field should be skipped during cooldown")
	}
	if !got[2] {
		t.Fatal("never-synced comic should always be selected")
	}
	if !got[3] {
		t.Fatal("comic synced past the cooldown window should be selected again")
	}
	if !got[4] {
		t.Fatal("comic missing only a Comic Vine ID should be selected")
	}
	if !got[5] {
		t.Fatal("comic with only a Comic Vine ID should be selected")
	}
	if got[6] {
		t.Fatal("recently checked comic with only a Comic Vine ID should be skipped during cooldown")
	}
}

func TestMetronComicScanUsesSelectedIncompleteFields(t *testing.T) {
	db := newMetronComicScannerTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE comics (
			id INTEGER PRIMARY KEY,
			publisher TEXT NOT NULL DEFAULT '',
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			metron_issue_id INTEGER,
			comic_vine_id INTEGER,
			metron_synced_at TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO comics VALUES
			(1, '', '2020-01-01', '/one.jpg', 'Complete', 101, 1001, ''),
			(2, 'Publisher', '2020-01-01', '/two.jpg', 'Complete', 102, NULL, ''),
			(3, 'Publisher', '2020-01-01', '/three.jpg', '', 103, 1003, '');
	`); err != nil {
		t.Fatal(err)
	}

	settings := defaultMetronComicScanSettings()
	settings.RecheckCooldownDays = 0
	settings.IncompleteFields = []string{"publisher"}
	rows, err := selectIncompleteComics(context.Background(), db, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].MetronID.Int64 != 101 {
		t.Fatalf("publisher rows = %+v; want only Metron issue 101", rows)
	}

	settings.IncompleteFields = []string{"comicVineId", "description"}
	rows, err = selectIncompleteComics(context.Background(), db, settings, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].MetronID.Int64 != 102 || rows[1].MetronID.Int64 != 103 {
		t.Fatalf("selected rows = %+v; want Metron issues 102 and 103", rows)
	}
}

func TestMetronMaintenanceSelectsIncompleteLinkedResourcesAndRespectsCooldown(t *testing.T) {
	db := newMetronImportTestDB(t)
	now := time.Now().UTC()
	recent := now.Add(-time.Hour).Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT INTO characters (id, name, description, image, metron_character_id)
		VALUES (1, 'Missing aliases', 'Complete', '/character.jpg', 101);
		INSERT INTO series (id, name, series_year, publisher, volume, year_end, issue_count, description, metron_series_id)
		VALUES (1, 'Missing publisher', 2020, '', 2, 2024, 12, 'Complete', 201);
		INSERT INTO arcs (id, name, description, image, metron_arc_id)
		VALUES (1, 'Missing image', 'Complete', '', 301);
		INSERT INTO metron_sync_states (resource_type, metron_id, fully_synced, synced_at)
		VALUES ('series', 201, 1, ?);
	`, recent); err != nil {
		t.Fatalf("seed linked resources: %v", err)
	}

	settings := defaultMetronComicScanSettings()
	settings.RecheckCooldownDays = 30
	settings.CharacterIncompleteFields = []string{"aliases"}
	settings.SeriesIncompleteFields = []string{"publisher"}
	settings.ArcIncompleteFields = []string{"image"}

	characters, err := selectIncompleteCharacters(context.Background(), db, settings, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(characters) != 1 || characters[0].MetronID != 101 {
		t.Fatalf("characters = %+v; want Metron character 101", characters)
	}
	series, err := selectIncompleteSeries(context.Background(), db, settings, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 0 {
		t.Fatalf("recently checked series = %+v; want none during cooldown", series)
	}
	arcs, err := selectIncompleteArcs(context.Background(), db, settings, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(arcs) != 1 || arcs[0].MetronID != 301 {
		t.Fatalf("arcs = %+v; want Metron arc 301", arcs)
	}

	series, err = selectIncompleteSeries(context.Background(), db, settings, now.Add(31*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 1 || series[0].MetronID != 201 {
		t.Fatalf("series after cooldown = %+v; want Metron series 201", series)
	}
}

func TestMetronMaintenanceEnrichesLinkedResourceMetadataWithoutOverwritingLocalValues(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO characters (id, name, description, image, metron_character_id)
		VALUES (1, 'Local Character', '', '/keep-character.jpg', 101);
		INSERT INTO series (id, name, series_year, publisher, volume, year_end, issue_count, description, metron_series_id)
		VALUES (1, 'Local Series', 2020, 'Keep Publisher', 0, 0, 0, '', 201);
		INSERT INTO arcs (id, name, description, image, metron_arc_id)
		VALUES (1, 'Local Arc', '', '/keep-arc.jpg', 301);
	`); err != nil {
		t.Fatalf("seed linked resources: %v", err)
	}

	if err := enrichIncompleteCharacterFromMetron(ctx, db, nil, 1, metron.MetronCharacter{
		ID: 101, Name: "Remote Character", Description: "Remote description",
		Image: "/replace-character.jpg", Aliases: []string{"Hero"},
	}); err != nil {
		t.Fatalf("enrich character: %v", err)
	}
	if err := enrichIncompleteSeriesFromMetron(ctx, db, 1, metron.Series{
		ID: 201, Name: "Remote Series", Publisher: "Replace Publisher", YearBegan: 2021,
		Volume: 2, YearEnd: 2025, IssueCount: 24, Description: "Remote description",
	}); err != nil {
		t.Fatalf("enrich series: %v", err)
	}
	if err := enrichIncompleteArcFromMetron(ctx, db, 1, metron.MetronArc{
		ID: 301, Name: "Remote Arc", Description: "Remote description", Image: "/replace-arc.jpg",
	}); err != nil {
		t.Fatalf("enrich arc: %v", err)
	}

	var character Character
	if err := db.Get(&character, `SELECT * FROM characters WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if character.Name != "Local Character" || character.Description != "Remote description" ||
		character.Image != "/keep-character.jpg" {
		t.Fatalf("character = %+v; want missing description only", character)
	}
	var aliases int
	if err := db.Get(&aliases, `SELECT COUNT(*) FROM character_aliases WHERE character_id = 1 AND alias = 'Hero'`); err != nil || aliases != 1 {
		t.Fatalf("character aliases = %d, err=%v; want Hero", aliases, err)
	}

	var gotSeries ComicSeries
	if err := db.Get(&gotSeries, `SELECT * FROM series WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if gotSeries.Name != "Local Series" || gotSeries.Publisher != "Keep Publisher" ||
		gotSeries.SeriesYear != 2020 || gotSeries.Volume != 2 || gotSeries.YearEnd != 2025 ||
		gotSeries.IssueCount != 24 || gotSeries.Description != "Remote description" {
		t.Fatalf("series = %+v; missing metadata was not filled safely", gotSeries)
	}

	var arc Arc
	if err := db.Get(&arc, `SELECT * FROM arcs WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if arc.Name != "Local Arc" || arc.Description != "Remote description" || arc.Image != "/keep-arc.jpg" {
		t.Fatalf("arc = %+v; want missing description only", arc)
	}
}

func TestMetronMaintenanceProcessesResourcesInConfiguredOrder(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO comics (id, series, issue, publisher, metron_issue_id)
		VALUES (1, 'Local Series', '1', '', 101);
		INSERT INTO characters (id, name, description, image, metron_character_id)
		VALUES (1, 'Local Character', '', '', 201);
		INSERT INTO series (id, name, series_year, publisher, metron_series_id)
		VALUES (1, 'Local Series', 2020, '', 301);
		INSERT INTO arcs (id, name, description, image, metron_arc_id)
		VALUES (1, 'Local Arc', '', '', 401);
	`); err != nil {
		t.Fatalf("seed maintenance resources: %v", err)
	}

	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/arc/401/":
			_, _ = w.Write([]byte(`{"id":401,"name":"Remote Arc","desc":"Arc description"}`))
		case "/series/301/":
			_, _ = w.Write([]byte(`{"id":301,"name":"Remote Series","publisher":{"name":"Publisher"},"year_began":2020}`))
		case "/character/201/":
			_, _ = w.Write([]byte(`{"id":201,"name":"Remote Character","desc":"Character description"}`))
		case "/issue/101/":
			_, _ = w.Write([]byte(`{"id":101,"number":"1","series":{"name":"Local Series","year_began":2020,"publisher":{"name":"Publisher"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.ScanCharacters = true
	settings.ScanSeries = true
	settings.ScanArcs = true
	settings.IncompleteFields = []string{"publisher"}
	settings.CharacterIncompleteFields = []string{"description"}
	settings.SeriesIncompleteFields = []string{"publisher"}
	settings.ArcIncompleteFields = []string{"description"}
	settings.ResourceOrder = []string{"arcs", "series", "characters", "comics"}
	settings.DailyCallLimit = 10
	settings.MinIntervalSeconds = 0

	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	want := []string{"/arc/401/", "/series/301/", "/character/201/", "/issue/101/"}
	if strings.Join(requests, ",") != strings.Join(want, ",") {
		t.Fatalf("request order = %v; want %v", requests, want)
	}
}

func TestMetronMaintenanceCanPullFullArcComicList(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO arcs (id, name, description, image, metron_arc_id)
		VALUES (1, 'Local Arc', '', '', 301);
	`); err != nil {
		t.Fatalf("seed arc: %v", err)
	}

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/arc/301/":
			_, _ = w.Write([]byte(`{"id":301,"name":"Remote Arc","desc":"Remote description"}`))
		case "/arc/301/issue_list/":
			_, _ = w.Write([]byte(`{"results":[{"issue":{"id":401,"number":"1","series":{"id":501,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}}}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.ScanComics = false
	settings.ScanArcs = true
	settings.PullArcComics = true
	settings.ArcIncompleteFields = []string{"description"}
	settings.DailyCallLimit = 10
	settings.MinIntervalSeconds = 0
	settings.RecheckCooldownDays = 30

	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	var description string
	if err := db.Get(&description, `SELECT description FROM arcs WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if description != "Remote description" {
		t.Fatalf("arc description = %q; want remote metadata", description)
	}
	var linked int
	if err := db.Get(&linked, `
		SELECT COUNT(*)
		FROM arc_comics ac
		JOIN comics c ON c.id = ac.comic_id
		WHERE ac.arc_id = 1 AND c.metron_issue_id = 401
	`); err != nil {
		t.Fatal(err)
	}
	if linked != 1 {
		t.Fatalf("linked arc comics = %d; want one imported comic", linked)
	}
	if requests["/arc/301/"] != 1 || requests["/arc/301/issue_list/"] != 1 {
		t.Fatalf("requests = %#v; want metadata and full comic list", requests)
	}
	usage := currentMetronComicScanUsage(context.Background(), db, time.Now())
	if usage.Calls != 2 {
		t.Fatalf("Metron calls = %d; want metadata plus comic list", usage.Calls)
	}
}

func TestMetronMaintenanceMetadataOnlySkipsArcComicList(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO arcs (id, name, description, image, metron_arc_id)
		VALUES (1, 'Local Arc', '', '', 301);
	`); err != nil {
		t.Fatalf("seed arc: %v", err)
	}

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/arc/301/" {
			_, _ = w.Write([]byte(`{"id":301,"name":"Remote Arc","desc":"Remote description"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.ScanComics = false
	settings.ScanArcs = true
	settings.PullArcComics = false
	settings.ArcIncompleteFields = []string{"description"}
	settings.DailyCallLimit = 10
	settings.MinIntervalSeconds = 0

	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	if requests["/arc/301/"] != 1 || requests["/arc/301/issue_list/"] != 0 {
		t.Fatalf("requests = %#v; metadata-only maintenance must not pull the comic list", requests)
	}
	var comics int
	if err := db.Get(&comics, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatal(err)
	}
	if comics != 0 {
		t.Fatalf("comics = %d; metadata-only maintenance imported comics", comics)
	}
	usage := currentMetronComicScanUsage(context.Background(), db, time.Now())
	if usage.Calls != 1 {
		t.Fatalf("Metron calls = %d; want metadata only", usage.Calls)
	}
}

func TestMetronMaintenanceQuotaAppliesToEveryComicListPage(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO arcs (id, name, description, image, metron_arc_id)
		VALUES (1, 'Local Arc', '', '', 301);
	`); err != nil {
		t.Fatalf("seed arc: %v", err)
	}

	requests := map[string]int{}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.String()]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.String() {
		case "/arc/301/":
			_, _ = w.Write([]byte(`{"id":301,"name":"Remote Arc","desc":"Remote description"}`))
		case "/arc/301/issue_list/":
			_, _ = w.Write([]byte(`{"count":2,"next":"` + server.URL + `/arc/301/issue_list/?page=2","results":[{"issue":{"id":401,"number":"1","series":{"name":"Series"}}}]}`))
		case "/arc/301/issue_list/?page=2":
			_, _ = w.Write([]byte(`{"count":2,"next":null,"results":[{"issue":{"id":402,"number":"2","series":{"name":"Series"}}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.ScanComics = false
	settings.ScanArcs = true
	settings.PullArcComics = true
	settings.ArcIncompleteFields = []string{"description"}
	settings.DailyCallLimit = 2
	settings.MinIntervalSeconds = 0

	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	if requests["/arc/301/"] != 1 || requests["/arc/301/issue_list/"] != 1 ||
		requests["/arc/301/issue_list/?page=2"] != 0 {
		t.Fatalf("requests = %#v; quota should stop before the second list page", requests)
	}
	status := scanner.snapshot(context.Background())
	if status.StopReason != "daily quota used" || status.CallsUsedToday != 2 {
		t.Fatalf("status = %+v; want exhausted two-call quota", status)
	}
}

func TestMetronComicScanFindsAndCoolsDownComicVineOnlyComics(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO comics (series, issue, publisher, comic_vine_id) VALUES
			('Found', '1', '', 9001),
			('Not found', '2', '', 9002)
	`); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.String() {
		case "/issue/?cv_id=9001":
			_, _ = w.Write([]byte(`{"results":[{"id":77,"cv_id":9001}]}`))
		case "/issue/77/":
			_, _ = w.Write([]byte(`{"id":77,"cv_id":9001,"number":"1","series":{"name":"Found","year_began":2026,"publisher":{"name":"Publisher"}}}`))
		case "/issue/?cv_id=9002":
			_, _ = w.Write([]byte(`{"results":[]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.DailyCallLimit = 10
	settings.MinIntervalSeconds = 0
	settings.RecheckCooldownDays = 30
	settings.IncompleteFields = []string{"publisher"}
	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	var found struct {
		MetronID  int    `db:"metron_issue_id"`
		Publisher string `db:"publisher"`
	}
	if err := db.Get(&found, `SELECT metron_issue_id, publisher FROM comics WHERE comic_vine_id = 9001`); err != nil {
		t.Fatal(err)
	}
	if found.MetronID != 77 || found.Publisher != "Publisher" {
		t.Fatalf("found comic = %+v; want Metron ID 77 and enriched publisher", found)
	}

	var checkedAt string
	if err := db.Get(&checkedAt, `SELECT metron_synced_at FROM comics WHERE comic_vine_id = 9002`); err != nil {
		t.Fatal(err)
	}
	if checkedAt == "" {
		t.Fatal("unmatched Comic Vine comic was not marked as checked")
	}

	now := time.Now()
	rows, err := selectIncompleteComics(context.Background(), db, settings, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows during cooldown = %+v; want none", rows)
	}
	rows, err = selectIncompleteComics(context.Background(), db, settings, now.Add(31*24*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].ComicVineID.Int64 != 9002 {
		t.Fatalf("rows after cooldown = %+v; want Comic Vine ID 9002", rows)
	}

	usage := currentMetronComicScanUsage(context.Background(), db, time.Now())
	if usage.Calls != 3 {
		t.Fatalf("Metron calls = %d; want two searches and one detail request", usage.Calls)
	}
}

func TestMetronComicScanSkipsComicVineMismatch(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
		INSERT INTO comics (series, issue, publisher, metron_issue_id, comic_vine_id)
		VALUES ('Local', '1', '', 77, 9001)
	`); err != nil {
		t.Fatalf("seed comic: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.String() == "/issue/77/" {
			_, _ = w.Write([]byte(`{"id":77,"cv_id":9999,"number":"1","series":{"name":"Metron","publisher":{"name":"Publisher"}}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.DailyCallLimit = 10
	settings.MinIntervalSeconds = 0
	settings.IncompleteFields = []string{"publisher"}
	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	var comic struct {
		Publisher string `db:"publisher"`
		SyncedAt  string `db:"metron_synced_at"`
	}
	if err := db.Get(&comic, `SELECT publisher, metron_synced_at FROM comics WHERE metron_issue_id = 77`); err != nil {
		t.Fatal(err)
	}
	if comic.Publisher != "" {
		t.Fatalf("mismatched comic was enriched with publisher %q", comic.Publisher)
	}
	if comic.SyncedAt == "" {
		t.Fatal("mismatched comic was not marked as checked")
	}
}

func TestEnrichIncompleteComicStoresComicVineID(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, metron_issue_id)
		VALUES ('Series', 2026, '1', 'Publisher', 77)
	`); err != nil {
		t.Fatalf("seed comic: %v", err)
	}
	if err := enrichIncompleteComicFromMetron(ctx, db, nil, 1, metron.Issue{
		ID:          77,
		ComicVineID: 9988,
		Series:      "Series",
		SeriesYear:  2026,
		Issue:       "1",
		Publisher:   "Publisher",
	}); err != nil {
		t.Fatalf("enrich comic: %v", err)
	}
	var comicVineID int
	if err := db.Get(&comicVineID, `SELECT comic_vine_id FROM comics WHERE id = 1`); err != nil {
		t.Fatalf("read Comic Vine ID: %v", err)
	}
	if comicVineID != 9988 {
		t.Fatalf("Comic Vine ID = %d; want 9988", comicVineID)
	}
}

func TestEnrichIncompleteComicSkipsConflictingComicVineID(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (1, 'Target', 2026, '1', '', 77);
		INSERT INTO comics (id, series, series_year, issue, publisher, comic_vine_id)
		VALUES (2, 'Existing Comic Vine row', 2026, '1', 'Publisher', 9988);
	`); err != nil {
		t.Fatalf("seed comics: %v", err)
	}

	if err := enrichIncompleteComicFromMetron(ctx, db, nil, 1, metron.Issue{
		ID:          77,
		ComicVineID: 9988,
		Publisher:   "Publisher",
	}); err != nil {
		t.Fatalf("enrich comic with conflicting Comic Vine ID: %v", err)
	}

	var comic struct {
		ComicVineID *int   `db:"comic_vine_id"`
		Publisher   string `db:"publisher"`
	}
	if err := db.Get(&comic, `SELECT comic_vine_id, publisher FROM comics WHERE id = 1`); err != nil {
		t.Fatalf("read enriched comic: %v", err)
	}
	if comic.ComicVineID != nil {
		t.Fatalf("conflicting Comic Vine ID was attached: %d", *comic.ComicVineID)
	}
	if comic.Publisher != "Publisher" {
		t.Fatalf("publisher = %q; want metadata enrichment to succeed", comic.Publisher)
	}
}

func TestEnrichIncompleteComicSkipsConflictingMetronID(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (id, series, series_year, issue, publisher, comic_vine_id)
		VALUES (1, 'Target', 2026, '1', '', 9988);
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (2, 'Existing Metron row', 2026, '1', 'Publisher', 77);
	`); err != nil {
		t.Fatalf("seed comics: %v", err)
	}

	if err := enrichIncompleteComicFromMetron(ctx, db, nil, 1, metron.Issue{
		ID:          77,
		ComicVineID: 9988,
		Publisher:   "Publisher",
	}); err != nil {
		t.Fatalf("enrich comic with conflicting Metron ID: %v", err)
	}

	var comic struct {
		MetronID  *int   `db:"metron_issue_id"`
		Publisher string `db:"publisher"`
	}
	if err := db.Get(&comic, `SELECT metron_issue_id, publisher FROM comics WHERE id = 1`); err != nil {
		t.Fatalf("read enriched comic: %v", err)
	}
	if comic.MetronID != nil {
		t.Fatalf("conflicting Metron ID was attached: %d", *comic.MetronID)
	}
	if comic.Publisher != "Publisher" {
		t.Fatalf("publisher = %q; want metadata enrichment to succeed", comic.Publisher)
	}
}

func TestMetronComicScanReportsLastFailure(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		CREATE TABLE app_settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
		INSERT INTO comics (id, series, issue, publisher, metron_issue_id)
		VALUES (1, 'Target', '1', '', 77);
	`); err != nil {
		t.Fatalf("seed scanner data: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "Metron unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	settings := defaultMetronComicScanSettings()
	settings.DailyCallLimit = 10
	settings.MinIntervalSeconds = 0
	settings.RecheckCooldownDays = 0
	settings.IncompleteFields = []string{"publisher"}
	scanner := NewMetronComicScanner(db, metron.New(metron.Config{BaseURL: server.URL}), nil)
	scanner.run(context.Background(), settings)

	status := scanner.snapshot(context.Background())
	if status.Failed != 1 {
		t.Fatalf("failed = %d; want 1", status.Failed)
	}
	if !strings.Contains(status.LastError, "comic 1: fetch Metron issue") {
		t.Fatalf("last error = %q; want comic and failure stage", status.LastError)
	}
}

func TestComicScanIntervalIsScopedToOneRun(t *testing.T) {
	var firstRunNext time.Time
	if err := waitForComicScanInterval(context.Background(), &firstRunNext, time.Second); err != nil {
		t.Fatal(err)
	}
	if firstRunNext.IsZero() {
		t.Fatal("first scan run did not record its next request time")
	}

	var otherRunNext time.Time
	started := time.Now()
	if err := waitForComicScanInterval(context.Background(), &otherRunNext, time.Second); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed > 100*time.Millisecond {
		t.Fatalf("independent scan run waited %v for another run", elapsed)
	}
}
