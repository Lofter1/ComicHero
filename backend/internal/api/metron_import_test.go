package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func newMetronImportTestDB(t *testing.T) *sqlx.DB {
	t.Helper()

	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE reading_orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_reading_list_id INTEGER,
			author_user_id INTEGER REFERENCES users(id) ON DELETE SET NULL
		);
		CREATE UNIQUE INDEX idx_reading_orders_metron_reading_list_id
		ON reading_orders(metron_reading_list_id)
		WHERE metron_reading_list_id IS NOT NULL;

		CREATE TABLE comics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			series TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			issue TEXT NOT NULL,
			publisher TEXT NOT NULL,
			cover_date TEXT NOT NULL DEFAULT '',
			cover_image TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			read INTEGER NOT NULL DEFAULT 0,
			metron_issue_id INTEGER
		);
		CREATE UNIQUE INDEX idx_comics_metron_issue_id
		ON comics(metron_issue_id)
		WHERE metron_issue_id IS NOT NULL;

		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			is_default INTEGER NOT NULL DEFAULT 0,
			is_admin INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE user_comics (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			read INTEGER NOT NULL DEFAULT 0,
			read_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (comic_id, user_id)
		);
		INSERT OR IGNORE INTO users (id, name, is_default) VALUES (1, 'Default', 1);

		CREATE TABLE series (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			series_year INTEGER NOT NULL DEFAULT 0,
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_series_id INTEGER,
			publisher TEXT NOT NULL DEFAULT '',
			volume INTEGER NOT NULL DEFAULT 0,
			year_end INTEGER NOT NULL DEFAULT 0,
			issue_count INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX idx_series_name_year
		ON series(name, series_year);
		CREATE UNIQUE INDEX idx_series_metron_series_id
		ON series(metron_series_id)
		WHERE metron_series_id IS NOT NULL;

		CREATE TABLE characters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_character_id INTEGER
		);
		CREATE UNIQUE INDEX idx_characters_metron_character_id
		ON characters(metron_character_id)
		WHERE metron_character_id IS NOT NULL;

		CREATE TABLE character_aliases (
			character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
			alias TEXT NOT NULL,
			PRIMARY KEY (character_id, alias)
		);

		CREATE TABLE comic_characters (
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			character_id INTEGER NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
			PRIMARY KEY (comic_id, character_id)
		);

		CREATE TABLE arcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0,
			metron_arc_id INTEGER,
			image TEXT NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX idx_arcs_metron_arc_id
		ON arcs(metron_arc_id)
		WHERE metron_arc_id IS NOT NULL;

		CREATE TABLE arc_comics (
			arc_id INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);

		CREATE TABLE reading_order_comics (
			reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT ''
		);

		CREATE TABLE reading_order_children (
			parent_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			child_reading_order_id INTEGER NOT NULL REFERENCES reading_orders(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (parent_reading_order_id, child_reading_order_id),
			CHECK (parent_reading_order_id <> child_reading_order_id)
		);

		CREATE TABLE metron_sync_states (
			resource_type TEXT NOT NULL,
			metron_id INTEGER NOT NULL,
			last_modified TEXT NOT NULL DEFAULT '',
			fully_synced INTEGER NOT NULL DEFAULT 0,
			synced_at TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (resource_type, metron_id)
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func serverNextURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}

func TestImportMetronComicReusesExistingMetronComic(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	issue := metron.Issue{
		ID:         101,
		Series:     "Series",
		SeriesYear: 2026,
		Issue:      "1",
		Publisher:  "Publisher",
	}

	first, err := importMetronComic(ctx, db, nil, nil, issue)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := importMetronComic(ctx, db, nil, nil, issue)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if first.Body.ID != second.Body.ID {
		t.Fatalf("comic ids differ: first=%d second=%d", first.Body.ID, second.Body.ID)
	}

	var count int
	if err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if count != 1 {
		t.Fatalf("comic count = %d; want 1", count)
	}
}

func TestMetronConditionalRequestRequiresFullSyncState(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	if _, err := db.ExecContext(ctx, `
		INSERT INTO comics (series, series_year, issue, publisher, metron_issue_id)
		VALUES ('Series', 2026, '1', 'Publisher', 123)
	`); err != nil {
		t.Fatalf("insert partial comic: %v", err)
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			if got := r.Header.Get("If-Modified-Since"); got != "" {
				t.Fatalf("first If-Modified-Since = %q; want empty", got)
			}
			w.Header().Set("Last-Modified", "Wed, 12 Feb 2026 10:30:00 GMT")
			w.Write([]byte(`{"id":123,"series":{"name":"Series","year_began":2026},"number":"1","cover_date":"2026-01-01"}`))
		case 2:
			if got := r.Header.Get("If-Modified-Since"); got != "Wed, 12 Feb 2026 10:30:00 GMT" {
				t.Fatalf("second If-Modified-Since = %q; want stored Last-Modified", got)
			}
			w.WriteHeader(http.StatusNotModified)
		case 3:
			if got := r.Header.Get("If-Modified-Since"); got != "" {
				t.Fatalf("forced If-Modified-Since = %q; want empty", got)
			}
			w.Header().Set("Last-Modified", "Thu, 13 Feb 2026 10:30:00 GMT")
			w.Write([]byte(`{"id":123,"series":{"name":"Series","year_began":2026},"number":"1","cover_date":"2026-01-02"}`))
		default:
			t.Fatalf("unexpected request %d", requests)
		}
	}))
	defer server.Close()

	client := metron.New(metron.Config{BaseURL: server.URL})
	issue, info, err := fetchMetronIssue(ctx, db, client, 123, false)
	if err != nil {
		t.Fatalf("fetchMetronIssue first: %v", err)
	}
	if issue == nil || issue.ID != 123 || info.NotModified {
		t.Fatalf("first issue/info = %#v/%#v", issue, info)
	}
	if err := markMetronSynced(ctx, db, metronResourceIssue, 123, info); err != nil {
		t.Fatalf("markMetronSynced: %v", err)
	}

	issue, info, err = fetchMetronIssue(ctx, db, client, 123, false)
	if err != nil {
		t.Fatalf("fetchMetronIssue second: %v", err)
	}
	if issue != nil || !info.NotModified {
		t.Fatalf("second issue/info = %#v/%#v; want not modified", issue, info)
	}

	issue, info, err = fetchMetronIssue(ctx, db, client, 123, true)
	if err != nil {
		t.Fatalf("fetchMetronIssue forced: %v", err)
	}
	if issue == nil || issue.CoverDate != "2026-01-02" || info.NotModified {
		t.Fatalf("forced issue/info = %#v/%#v", issue, info)
	}
}

func TestImportMetronComicPreservesIssueNumberSuffix(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	comic, err := importMetronComic(ctx, db, nil, nil, metron.Issue{
		ID:         18030,
		Series:     "The Amazing Spider-Man",
		SeriesYear: 2018,
		Issue:      "50.LR",
		Number:     "50.LR",
		Publisher:  "Marvel",
		CoverDate:  "2020-12-01",
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if comic.Body.Issue != "50.LR" {
		t.Fatalf("issue = %q, want 50.LR", comic.Body.Issue)
	}
	if comic.Body.Title != "The Amazing Spider-Man (2018) #50.LR" {
		t.Fatalf("title = %q, want suffix in title", comic.Body.Title)
	}

	var storedIssue string
	if err := db.GetContext(ctx, &storedIssue, `SELECT issue FROM comics WHERE id = ?`, comic.Body.ID); err != nil {
		t.Fatalf("stored issue: %v", err)
	}
	if storedIssue != "50.LR" {
		t.Fatalf("stored issue = %q, want 50.LR", storedIssue)
	}
}

func TestImportMetronComicSavesCharacterAppearancesAndAliases(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	issue := metron.Issue{
		ID:         101,
		Series:     "Series",
		SeriesYear: 2026,
		Issue:      "1",
		Publisher:  "Publisher",
		Characters: []metron.MetronCharacter{
			{ID: 301, Name: "Hero", Aliases: []string{"The Hero"}},
		},
	}

	comic, err := importMetronComicWithOptions(ctx, db, nil, nil, issue, MetronImportOptions{Mode: "full"})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(comic.Body.Characters) != 1 {
		t.Fatalf("comic characters = %d; want 1", len(comic.Body.Characters))
	}

	characters, err := listCharacters(ctx, db, &CharacterListInput{Query: "Hero"})
	if err != nil {
		t.Fatalf("listCharacters: %v", err)
	}
	if len(characters.Body) != 1 {
		t.Fatalf("characters = %d; want 1", len(characters.Body))
	}
	if characters.Body[0].AppearanceCount != 1 {
		t.Fatalf("appearance count = %d; want 1", characters.Body[0].AppearanceCount)
	}
	if len(characters.Body[0].Aliases) != 1 || characters.Body[0].Aliases[0] != "The Hero" {
		t.Fatalf("aliases = %#v; want The Hero", characters.Body[0].Aliases)
	}

	detail, err := getCharacter(ctx, db, characters.Body[0].ID)
	if err != nil {
		t.Fatalf("getCharacter: %v", err)
	}
	if len(detail.Body.Comics) != 1 || detail.Body.Comics[0].ID != comic.Body.ID {
		t.Fatalf("appearances = %#v; want imported comic", detail.Body.Comics)
	}
}

func TestQuickImportMetronComicSavesArcRelationshipWithoutFetchingArc(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":501,"name":"Expanded Arc","desc":"Full metadata"}`))
	}))
	defer server.Close()

	client := metron.New(metron.Config{BaseURL: server.URL})
	comic, err := importMetronComic(ctx, db, client, nil, metron.Issue{
		ID:         101,
		Series:     "Series",
		SeriesYear: 2026,
		Issue:      "1",
		Publisher:  "Publisher",
		Arcs: []metron.MetronArc{
			{ID: 501, Name: "Payload Arc"},
		},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if requests["/arc/501/"] != 0 {
		t.Fatalf("fetched arc detail %d times; want 0", requests["/arc/501/"])
	}

	detail, err := getComic(ctx, db, comic.Body.ID)
	if err != nil {
		t.Fatalf("getComic: %v", err)
	}
	if len(detail.Body.Arcs) != 1 || detail.Body.Arcs[0].Name != "Payload Arc" {
		t.Fatalf("comic arcs = %#v; want payload arc", detail.Body.Arcs)
	}
}

func TestFullImportMetronComicExpandsArcMetadataWithoutIssueList(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/arc/501/":
			w.Write([]byte(`{"id":501,"name":"Expanded Arc","desc":"Full metadata","image":"https://example.test/arc.jpg"}`))
		case "/arc/501/issue_list/":
			w.Write([]byte(`{"results":[{"id":999,"series":{"name":"Other"},"number":"1"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := metron.New(metron.Config{BaseURL: server.URL})
	comic, err := importMetronComicWithOptions(ctx, db, client, nil, metron.Issue{
		ID:         101,
		Series:     "Series",
		SeriesYear: 2026,
		Issue:      "1",
		Publisher:  "Publisher",
		Arcs: []metron.MetronArc{
			{ID: 501, Name: "Payload Arc"},
		},
	}, MetronImportOptions{Mode: "full"})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if requests["/arc/501/"] != 1 {
		t.Fatalf("fetched arc detail %d times; want 1", requests["/arc/501/"])
	}
	if requests["/arc/501/issue_list/"] != 0 {
		t.Fatalf("fetched arc issue list %d times; want 0", requests["/arc/501/issue_list/"])
	}

	detail, err := getComic(ctx, db, comic.Body.ID)
	if err != nil {
		t.Fatalf("getComic: %v", err)
	}
	if len(detail.Body.Arcs) != 1 || detail.Body.Arcs[0].Description != "Full metadata" || detail.Body.Arcs[0].Image == "" {
		t.Fatalf("comic arcs = %#v; want expanded arc metadata", detail.Body.Arcs)
	}
}

func TestListCharactersReturnsFavoriteAndProgress(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO characters (id, name, favorite) VALUES (1, 'Hero', 1);
		INSERT INTO characters (id, name, favorite) VALUES (2, 'Villain', 0);
		INSERT INTO comics (id, series, issue, publisher, read) VALUES (1, 'Series', 1, 'Publisher', 1);
		INSERT INTO comics (id, series, issue, publisher, read) VALUES (2, 'Series', 2, 'Publisher', 0);
		INSERT INTO user_comics (comic_id, user_id, read) SELECT id, (SELECT id FROM users WHERE name = 'Default'), read FROM comics WHERE id IN (1, 2);
		INSERT INTO comic_characters (comic_id, character_id) VALUES (1, 1);
		INSERT INTO comic_characters (comic_id, character_id) VALUES (2, 1);
	`); err != nil {
		t.Fatalf("insert fixtures: %v", err)
	}

	characters, err := listCharacters(ctx, db, &CharacterListInput{Favorite: "true"})
	if err != nil {
		t.Fatalf("listCharacters: %v", err)
	}
	if len(characters.Body) != 1 {
		t.Fatalf("characters = %d; want 1 favorite", len(characters.Body))
	}
	if !characters.Body[0].Favorite {
		t.Fatal("favorite = false; want true")
	}
	if characters.Body[0].Progress != 0.5 {
		t.Fatalf("progress = %v; want 0.5", characters.Body[0].Progress)
	}
}

func TestImportMetronComicDoesNotFetchCharacterDetails(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":301,"name":"Hero","alias":["The Hero"]}`))
	}))
	defer server.Close()

	client := metron.New(metron.Config{BaseURL: server.URL})
	_, err := importMetronComic(ctx, db, client, nil, metron.Issue{
		ID:         101,
		Series:     "Series",
		SeriesYear: 2026,
		Issue:      "1",
		Publisher:  "Publisher",
		Characters: []metron.MetronCharacter{
			{ID: 301, Name: "Hero"},
		},
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if requests["/character/301/"] != 0 {
		t.Fatalf("fetched character detail %d times; want 0", requests["/character/301/"])
	}
}

func TestImportMetronComicSkipsExistingCharacterImport(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO characters (id, name, description, image, metron_character_id)
		VALUES (1, 'Original Hero', 'Keep this', '', 301)
	`); err != nil {
		t.Fatalf("insert character: %v", err)
	}

	comic, err := importMetronComicWithOptions(ctx, db, nil, nil, metron.Issue{
		ID:         101,
		Series:     "Series",
		SeriesYear: 2026,
		Issue:      "1",
		Publisher:  "Publisher",
		Characters: []metron.MetronCharacter{
			{ID: 301, Name: "Changed Hero", Aliases: []string{"New Alias"}},
		},
	}, MetronImportOptions{Mode: "full"})
	if err != nil {
		t.Fatalf("import: %v", err)
	}

	var character Character
	if err := db.GetContext(ctx, &character, `
		SELECT ch.*, COUNT(cc.comic_id) AS appearance_count
		FROM characters ch
		LEFT JOIN comic_characters cc ON cc.character_id = ch.id
		WHERE ch.id = 1
		GROUP BY ch.id
	`); err != nil {
		t.Fatalf("get character: %v", err)
	}
	if character.Name != "Original Hero" {
		t.Fatalf("character name = %q; want existing value", character.Name)
	}
	if character.AppearanceCount != 1 {
		t.Fatalf("appearance count = %d; want 1", character.AppearanceCount)
	}

	var aliasCount int
	if err := db.GetContext(ctx, &aliasCount, `SELECT COUNT(*) FROM character_aliases WHERE character_id = 1`); err != nil {
		t.Fatalf("count aliases: %v", err)
	}
	if aliasCount != 0 {
		t.Fatalf("alias count = %d; want 0", aliasCount)
	}
	if len(comic.Body.Characters) != 1 || comic.Body.Characters[0].Name != "Original Hero" {
		t.Fatalf("comic characters = %#v; want existing character", comic.Body.Characters)
	}
}

func TestImportCharacterAppearancesFromMetron(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO characters (id, name, description, image, favorite, metron_character_id)
		VALUES (1, 'Old Hero', 'Old description', 'old-image', 1, 301)
	`); err != nil {
		t.Fatalf("insert character: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (1, 'Series', 2026, 1, 'Publisher', 101)
	`); err != nil {
		t.Fatalf("insert existing comic: %v", err)
	}

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.String()]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.String() {
		case "/character/301/":
			w.Write([]byte(`{"id":301,"name":"Hero","description":"Fresh description","image":"fresh-image","aliases":["The Hero"]}`))
		case "/character/301/issue_list/":
			w.Write([]byte(`{
				"count": 2,
				"next": "` + serverNextURL(r, "/character/301/issue_list/?page=2") + `",
				"results": [
					{"id":101,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"1","cover_date":"2026-01-01"}
				]
			}`))
		case "/character/301/issue_list/?page=2":
			w.Write([]byte(`{
				"count": 2,
				"next": null,
				"results": [
					{"issue":{"id":102,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"2","cover_date":"2026-02-01"}}
				]
			}`))
		case "/issue/102/":
			w.Write([]byte(`{"id":102,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"2","cover_date":"2026-02-01"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := metron.New(metron.Config{BaseURL: server.URL})
	detail, err := importCharacterAppearancesFromMetron(ctx, db, client, nil, 1)
	if err != nil {
		t.Fatalf("importCharacterAppearancesFromMetron: %v", err)
	}
	if requests["/character/301/issue_list/"] != 1 {
		t.Fatalf("first issue-list page requests = %d; want 1", requests["/character/301/issue_list/"])
	}
	if requests["/character/301/"] != 1 {
		t.Fatalf("character detail requests = %d; want 1", requests["/character/301/"])
	}
	if requests["/character/301/issue_list/?page=2"] != 1 {
		t.Fatalf("second issue-list page requests = %d; want 1", requests["/character/301/issue_list/?page=2"])
	}
	if requests["/issue/101/"] != 0 {
		t.Fatalf("existing issue detail requests = %d; want 0", requests["/issue/101/"])
	}
	if requests["/issue/102/"] != 0 {
		t.Fatalf("new issue detail requests = %d; want 0", requests["/issue/102/"])
	}
	if len(detail.Body.Comics) != 2 {
		t.Fatalf("appearances = %d; want 2", len(detail.Body.Comics))
	}
	if detail.Body.Name != "Hero" || detail.Body.Description != "Fresh description" || detail.Body.Image != "fresh-image" {
		t.Fatalf("character metadata = %#v; want refreshed Metron metadata", detail.Body.Character)
	}
	if !detail.Body.Favorite {
		t.Fatal("character favorite was not preserved")
	}
	if len(detail.Body.Aliases) != 1 || detail.Body.Aliases[0] != "The Hero" {
		t.Fatalf("aliases = %#v; want The Hero", detail.Body.Aliases)
	}

	var linkCount int
	if err := db.GetContext(ctx, &linkCount, `SELECT COUNT(*) FROM comic_characters WHERE character_id = 1`); err != nil {
		t.Fatalf("count links: %v", err)
	}
	if linkCount != 2 {
		t.Fatalf("link count = %d; want 2", linkCount)
	}
}

func TestStartMetronCharacterAppearancesImport(t *testing.T) {
	db := newMetronImportTestDB(t)

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/character/301/":
			w.Write([]byte(`{"id":301,"name":"Hero"}`))
		case "/character/301/issue_list/":
			w.Write([]byte(`[{"id":101,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"1"}]`))
		case "/issue/101/":
			w.Write([]byte(`{"id":101,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := newMetronImportJobStore()
	client := metron.New(metron.Config{BaseURL: server.URL})
	job := startMetronCharacterAppearancesImport(testUserContext(), store, db, client, nil, 301)

	var current MetronImportJob
	for range 100 {
		var ok bool
		current, ok = store.get(job.ID)
		if !ok {
			t.Fatal("job not found")
		}
		if current.Status == "succeeded" || current.Status == "failed" || current.Status == "canceled" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current.Status != "succeeded" {
		t.Fatalf("job status = %q, message = %q; want succeeded", current.Status, current.Message)
	}
	if current.Type != "character" {
		t.Fatalf("job type = %q; want character", current.Type)
	}
	if requests["/character/301/issue_list/"] != 1 {
		t.Fatalf("issue list requests = %d; want 1", requests["/character/301/issue_list/"])
	}
}

func TestImportMetronReadingListReusesExistingOrderAndComics(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	list := metron.ReadingList{
		ID:          501,
		Name:        "Event",
		Description: "Big event",
		Image:       "https://example.test/event.jpg",
		Issues: []metron.Issue{
			{
				ID:         101,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      "1",
				Publisher:  "Publisher",
				Tags:       []string{"Main Story", "Tie-In"},
			},
		},
	}

	first, err := importMetronReadingList(ctx, db, nil, nil, list)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	second, err := importMetronReadingList(ctx, db, nil, nil, list)
	if err != nil {
		t.Fatalf("second import: %v", err)
	}
	if first.Body.ID != second.Body.ID {
		t.Fatalf("order ids differ: first=%d second=%d", first.Body.ID, second.Body.ID)
	}
	if second.Body.Image != "https://example.test/event.jpg" {
		t.Fatalf("image = %q; want Metron image", second.Body.Image)
	}

	var orderCount int
	if err := db.GetContext(ctx, &orderCount, `SELECT COUNT(*) FROM reading_orders`); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if orderCount != 1 {
		t.Fatalf("order count = %d; want 1", orderCount)
	}

	var comicCount int
	if err := db.GetContext(ctx, &comicCount, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if comicCount != 1 {
		t.Fatalf("comic count = %d; want 1", comicCount)
	}

	var tags string
	if err := db.GetContext(ctx, &tags, `SELECT tags FROM reading_order_comics WHERE reading_order_id = ?`, first.Body.ID); err != nil {
		t.Fatalf("select tags: %v", err)
	}
	if tags != "Main Story, Tie-In" {
		t.Fatalf("tags = %q; want Main Story, Tie-In", tags)
	}
}

func TestContinueMetronReadingListFillsExistingOrder(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO reading_orders (id, name, description, metron_reading_list_id)
		VALUES (1, 'Event', 'Partial import', 501)
	`); err != nil {
		t.Fatalf("insert incomplete reading order: %v", err)
	}

	err := continueMetronReadingListWithProgress(ctx, db, nil, nil, metron.ReadingList{
		ID:          501,
		Name:        "Event",
		Description: "Big event",
		Image:       "https://example.test/event.jpg",
		Issues: []metron.Issue{
			{
				ID:         101,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      "1",
				Publisher:  "Publisher",
			},
		},
	}, func(int, int, string) {})
	if err != nil {
		t.Fatalf("continueMetronReadingListWithProgress: %v", err)
	}

	var comicCount int
	if err := db.GetContext(ctx, &comicCount, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if comicCount != 1 {
		t.Fatalf("comic count = %d; want 1", comicCount)
	}
	var image string
	if err := db.GetContext(ctx, &image, `SELECT image FROM reading_orders WHERE id = 1`); err != nil {
		t.Fatalf("select image: %v", err)
	}
	if image != "https://example.test/event.jpg" {
		t.Fatalf("image = %q; want Metron image", image)
	}

	var linkedCount int
	if err := db.GetContext(ctx, &linkedCount, `SELECT COUNT(*) FROM reading_order_comics WHERE reading_order_id = 1`); err != nil {
		t.Fatalf("count linked comics: %v", err)
	}
	if linkedCount != 1 {
		t.Fatalf("linked comics = %d; want 1", linkedCount)
	}
}

func TestMetronReadingListLinksComicsDuringImportProgress(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	progressLinkedCounts := []int{}
	err := importMetronReadingListWithOptions(ctx, db, nil, nil, metron.ReadingList{
		ID:          501,
		Name:        "Event",
		Description: "Big event",
		Issues: []metron.Issue{
			{
				ID:         101,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      "1",
				Publisher:  "Publisher",
			},
			{
				ID:         102,
				Series:     "Series",
				SeriesYear: 2026,
				Issue:      "2",
				Publisher:  "Publisher",
			},
		},
	}, false, func(completed, total int, message string) {
		if completed <= 0 {
			return
		}
		var linkedCount int
		if err := db.GetContext(ctx, &linkedCount, `
			SELECT COUNT(*) FROM reading_order_comics roc
			JOIN reading_orders ro ON ro.id = roc.reading_order_id
			WHERE ro.metron_reading_list_id = 501
		`); err != nil {
			t.Fatalf("count linked comics during progress: %v", err)
		}
		progressLinkedCounts = append(progressLinkedCounts, linkedCount)
	}, defaultMetronImportOptions())
	if err != nil {
		t.Fatalf("importMetronReadingListWithOptions: %v", err)
	}

	if len(progressLinkedCounts) < 2 {
		t.Fatalf("progress linked counts = %#v; want counts during each issue", progressLinkedCounts)
	}
	if progressLinkedCounts[0] != 1 || progressLinkedCounts[1] != 2 {
		t.Fatalf("progress linked counts = %#v; want [1 2]", progressLinkedCounts)
	}
}

func TestUpdateComicReadStatus(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comics (id, series, series_year, issue, publisher, read)
		VALUES (1, 'Series', 2026, 1, 'Publisher', 0)
	`); err != nil {
		t.Fatalf("insert comic: %v", err)
	}

	detail, err := updateComicReadStatus(ctx, db, 1, true)
	if err != nil {
		t.Fatalf("updateComicReadStatus: %v", err)
	}
	if !detail.Body.Read {
		t.Fatal("comic read status was not updated")
	}
	if detail.Body.Title != "Series (2026) #1" {
		t.Fatalf("comic metadata changed unexpectedly: %#v", detail.Body)
	}

	var storedRead int
	if err := db.GetContext(ctx, &storedRead, `
		SELECT read FROM user_comics WHERE comic_id = ? AND user_id = (
			SELECT id FROM users WHERE name = 'Default'
		)
	`, 1); err != nil {
		t.Fatalf("read status row lookup: %v", err)
	}
	if storedRead != 1 {
		t.Fatalf("stored read flag = %d; want 1", storedRead)
	}
}

func TestImportMetronSeriesSkipsDetailFetchForExistingComic(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (1, 'Series', 2026, 1, 'Publisher', 101)
	`); err != nil {
		t.Fatalf("insert existing comic: %v", err)
	}

	detailRequests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		detailRequests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/issue/102/":
			w.Write([]byte(`{"id":102,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"2","cover_date":"2026-02-01"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := metron.New(metron.Config{BaseURL: server.URL})
	output, err := importMetronSeries(ctx, db, client, nil, []metron.Issue{
		{ID: 101, Series: "Series", SeriesYear: 2026, Issue: "1", Publisher: "Publisher"},
		{ID: 102, Series: "Series", SeriesYear: 2026, Issue: "2", Publisher: "Publisher"},
	})
	if err != nil {
		t.Fatalf("importMetronSeries: %v", err)
	}
	if len(output.Body) != 2 {
		t.Fatalf("imported comics = %d; want 2", len(output.Body))
	}
	if detailRequests["/issue/101/"] != 0 {
		t.Fatalf("fetched existing comic detail %d times; want 0", detailRequests["/issue/101/"])
	}
	if detailRequests["/issue/102/"] != 0 {
		t.Fatalf("fetched new comic detail %d times; want 0", detailRequests["/issue/102/"])
	}
}

func TestImportMetronSeriesOptionsControlDetailFetches(t *testing.T) {
	ctx := testUserContext()

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.Path]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/issue/201/":
			w.Write([]byte(`{"id":201,"series":{"id":401,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"1","characters":[{"id":301,"name":"Attached Hero"}]}`))
		case "/series/401/":
			w.Write([]byte(`{"id":401,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"},"issue_count":1}`))
		case "/character/301/":
			w.Write([]byte(`{"id":301,"name":"Full Hero","desc":"Full profile","alias":["Heroic"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	client := metron.New(metron.Config{BaseURL: server.URL})
	issues := []metron.Issue{{ID: 201, Series: "Series", SeriesYear: 2026, Issue: "1", Publisher: "Publisher"}}

	quickDB := newMetronImportTestDB(t)
	if _, err := importMetronSeriesWithProgressOptions(ctx, quickDB, client, nil, issues, func(int, int, string) {}, MetronImportOptions{Mode: "quick"}); err != nil {
		t.Fatalf("quick import: %v", err)
	}
	if requests["/issue/201/"] != 0 || requests["/series/401/"] != 0 || requests["/character/301/"] != 0 {
		t.Fatalf("quick requests = %#v; want no issue, series, or character detail calls", requests)
	}

	fullDB := newMetronImportTestDB(t)
	if _, err := importMetronSeriesWithProgressOptions(ctx, fullDB, client, nil, issues, func(int, int, string) {}, MetronImportOptions{Mode: "full"}); err != nil {
		t.Fatalf("full import: %v", err)
	}
	if requests["/issue/201/"] != 1 || requests["/series/401/"] != 1 || requests["/character/301/"] != 1 {
		t.Fatalf("full requests = %#v; want one issue, series, and character detail call", requests)
	}

	var character Character
	if err := fullDB.GetContext(ctx, &character, `SELECT * FROM characters WHERE metron_character_id = 301`); err != nil {
		t.Fatalf("full character: %v", err)
	}
	if character.Name != "Full Hero" || character.Description != "Full profile" {
		t.Fatalf("character = %#v; want full metadata", character)
	}
}

func TestImportLocalSeriesFromMetronUpdatesMetadataAndImportsMissingComics(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO series (id, name, series_year, favorite)
		VALUES (1, 'Series', 2026, 1);
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (1, 'Series', 2026, 1, 'Publisher', 101);
	`); err != nil {
		t.Fatalf("insert local series: %v", err)
	}

	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests[r.URL.String()]++
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.String() {
		case "/series/?name=Series":
			w.Write([]byte(`{"results":[{"id":405,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"},"volume":2,"year_end":2027,"issue_count":2,"description":"Series metadata"}]}`))
		case "/series/405/":
			w.Write([]byte(`{"id":405,"name":"Series","year_began":2026,"publisher":{"name":"Publisher"},"volume":2,"year_end":2027,"issue_count":2,"description":"Series metadata"}`))
		case "/series/405/issue_list/":
			w.Write([]byte(`[
				{"id":101,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"1"},
				{"id":102,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"2"}
			]`))
		case "/issue/102/":
			w.Write([]byte(`{"id":102,"series":{"name":"Series","year_began":2026,"publisher":{"name":"Publisher"}},"number":"2"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store := newMetronImportJobStore()
	client := metron.New(metron.Config{BaseURL: server.URL})
	output, err := importLocalSeriesFromMetron(ctx, db, client, nil, store, 1)
	if err != nil {
		t.Fatalf("importLocalSeriesFromMetron: %v", err)
	}

	var current MetronImportJob
	for range 100 {
		var ok bool
		current, ok = store.get(output.Body.ID)
		if !ok {
			t.Fatal("job not found")
		}
		if current.Status == "succeeded" || current.Status == "failed" || current.Status == "canceled" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if current.Status != "succeeded" {
		t.Fatalf("job status = %q, message = %q; want succeeded", current.Status, current.Message)
	}
	if requests["/issue/101/"] != 0 {
		t.Fatalf("existing issue detail requests = %d; want 0", requests["/issue/101/"])
	}
	if requests["/issue/102/"] != 0 {
		t.Fatalf("new issue detail requests = %d; want 0", requests["/issue/102/"])
	}

	var series ComicSeries
	if err := db.GetContext(ctx, &series, `SELECT * FROM series WHERE id = 1`); err != nil {
		t.Fatalf("get series: %v", err)
	}
	if series.MetronSeriesID == nil || *series.MetronSeriesID != 405 {
		t.Fatalf("metron series id = %#v; want 405", series.MetronSeriesID)
	}
	if series.Description != "Series metadata" || series.IssueCount != 2 || series.Volume != 2 {
		t.Fatalf("series metadata = %#v; want Metron metadata", series)
	}
	if !series.Favorite {
		t.Fatal("series favorite was not preserved")
	}

	var comicCount int
	if err := db.GetContext(ctx, &comicCount, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if comicCount != 2 {
		t.Fatalf("comic count = %d; want 2", comicCount)
	}
}

func TestUpdateSeriesMetronMetadataKeepsLocalRowOnNameYearConflict(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)

	if _, err := db.ExecContext(ctx, `
		INSERT INTO series (id, name, series_year)
		VALUES (1, 'Local Series', 1976);
		INSERT INTO series (id, name, series_year)
		VALUES (2, 'Metron Series', 1997);
	`); err != nil {
		t.Fatalf("insert series: %v", err)
	}

	err := updateSeriesMetronMetadata(ctx, db, 1, metron.Series{
		ID:          405,
		Name:        "Metron Series",
		YearBegan:   1997,
		Publisher:   "Publisher",
		Volume:      2,
		YearEnd:     2000,
		IssueCount:  259,
		Description: "Metadata",
	})
	if err != nil {
		t.Fatalf("updateSeriesMetronMetadata: %v", err)
	}

	var series ComicSeries
	if err := db.GetContext(ctx, &series, `SELECT * FROM series WHERE id = 1`); err != nil {
		t.Fatalf("get series: %v", err)
	}
	if series.Name != "Local Series" || series.SeriesYear != 1976 {
		t.Fatalf("series identity = %s %d; want local identity preserved", series.Name, series.SeriesYear)
	}
	if series.Publisher != "Publisher" || series.IssueCount != 259 || series.Description != "Metadata" {
		t.Fatalf("series metadata = %#v; want partial metadata applied", series)
	}
}
