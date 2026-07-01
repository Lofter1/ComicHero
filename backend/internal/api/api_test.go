package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
)

func TestParseOptionalBool(t *testing.T) {
	value, ok, err := parseOptionalBool("", "favorite")
	if err != nil || ok || value {
		t.Fatalf("empty value = %v, %v, %v; want false, false, nil", value, ok, err)
	}

	value, ok, err = parseOptionalBool("true", "favorite")
	if err != nil || !ok || !value {
		t.Fatalf("true value = %v, %v, %v; want true, true, nil", value, ok, err)
	}

	if _, _, err := parseOptionalBool("sometimes", "favorite"); err == nil {
		t.Fatal("invalid bool returned nil error")
	}
}

func TestPaginationHelpers(t *testing.T) {
	query, args, limit, offset := paginatedQuery("SELECT * FROM comics", []any{"arg"}, 250, -12)
	if query != "SELECT * FROM comics LIMIT ? OFFSET ?" {
		t.Fatalf("query = %q", query)
	}
	if limit != maxPageLimit || offset != 0 {
		t.Fatalf("limit/offset = %d/%d; want %d/0", limit, offset, maxPageLimit)
	}
	if len(args) != 3 || args[1] != maxPageLimit+1 || args[2] != 0 {
		t.Fatalf("args = %#v; want original arg plus page limit+1 and offset", args)
	}

	items, headers := pageItems([]int{1, 2, 3}, 2, 10, 42)
	if len(items) != 2 || items[0] != 1 || items[1] != 2 {
		t.Fatalf("items = %#v; want first two", items)
	}
	if headers.PageLimit != "2" || headers.PageOffset != "10" || headers.HasMore != "true" || headers.TotalCount != "42" {
		t.Fatalf("headers = %#v; want limit 2, offset 10, has more true, total 42", headers)
	}
}

func TestComicListQuery(t *testing.T) {
	query, args, err := comicListQuery(&ComicListInput{
		Query:          "bat",
		Series:         "Detective",
		Publisher:      "DC",
		Read:           "false",
		ReadingOrderID: 12,
	})
	if err != nil {
		t.Fatalf("comicListQuery returned error: %v", err)
	}

	for _, fragment := range []string{
		"c.series LIKE ?",
		"c.series_year AS TEXT",
		"c.issue AS TEXT",
		"c.publisher LIKE ?",
		"c.read = ?",
		"roc.reading_order_id = ?",
		"ORDER BY c.series, c.series_year, CAST(c.issue AS REAL), c.issue",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query missing %q: %s", fragment, query)
		}
	}
	if len(args) != 10 {
		t.Fatalf("len(args) = %d; want 10", len(args))
	}
}

func TestReadingOrderHelpers(t *testing.T) {
	progress := computeProgress([]ReadingOrderComic{
		{Comic: Comic{Read: true}},
		{Comic: Comic{Read: false}},
	})
	if progress != 0.5 {
		t.Fatalf("progress = %v; want 0.5", progress)
	}

	input := &SetReadingOrderComicsInput{}
	input.Body.ComicIDs = []int{1, 2}

	items := readingOrderComicItems(input)
	if len(items) != 2 || items[0].ComicID != 1 || items[1].ComicID != 2 {
		t.Fatalf("items = %#v; want comic IDs 1, 2", items)
	}

	ids := readingOrderComicIDs([]ReadingOrderComicPayload{
		{ComicID: 3},
		{ComicID: 3},
	})
	if len(ids) != 2 || ids[0] != 3 || ids[1] != 3 {
		t.Fatalf("ids = %#v; want duplicate IDs preserved", ids)
	}
}

func TestArcCreateEntriesFavoriteAndProgress(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE arcs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			metron_arc_id INTEGER,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			image TEXT NOT NULL DEFAULT '',
			favorite INTEGER NOT NULL DEFAULT 0
		);
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
		CREATE TABLE arc_comics (
			arc_id INTEGER NOT NULL REFERENCES arcs(id) ON DELETE CASCADE,
			comic_id INTEGER NOT NULL REFERENCES comics(id) ON DELETE CASCADE,
			position INTEGER NOT NULL DEFAULT 0,
			note TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO comics (series, series_year, issue, publisher, read)
		VALUES ('Series', 2026, 1, 'Publisher', 1),
			('Series', 2026, 2, 'Publisher', 0);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	created, err := createArc(ctx, db, ArcPayload{Name: "Arc", Description: "Story"})
	if err != nil {
		t.Fatalf("createArc: %v", err)
	}

	input := &SetArcComicsInput{ID: created.Body.ID}
	input.Body.Comics = []ArcComicPayload{
		{ComicID: 1, Comment: "Start"},
		{ComicID: 1, Comment: "Again"},
		{ComicID: 2, Comment: "End"},
	}
	detail, err := setArcComics(ctx, db, input)
	if err != nil {
		t.Fatalf("setArcComics: %v", err)
	}
	if len(detail.Body.Comics) != 3 {
		t.Fatalf("arc comics = %d; want duplicate-preserving count 3", len(detail.Body.Comics))
	}
	if detail.Body.Progress != float64(2)/float64(3) {
		t.Fatalf("progress = %v; want 2/3", detail.Body.Progress)
	}
	if detail.Body.Comics[1].Comment != "Again" {
		t.Fatalf("second comment = %q; want Again", detail.Body.Comics[1].Comment)
	}

	updated, err := updateArc(ctx, db, created.Body.ID, ArcPayload{Name: "Arc", Description: "Story", Favorite: true})
	if err != nil {
		t.Fatalf("updateArc: %v", err)
	}
	if !updated.Body.Favorite {
		t.Fatal("arc favorite was not saved")
	}

	list, err := listArcs(ctx, db, &ArcListInput{ComicID: 2})
	if err != nil {
		t.Fatalf("listArcs: %v", err)
	}
	if len(list.Body) != 1 || list.Body[0].ID != created.Body.ID {
		t.Fatalf("filtered arcs = %#v; want created arc", list.Body)
	}
}

func TestSeriesFavoriteAndProgress(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
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
		INSERT INTO comics (series, series_year, issue, publisher, read)
		VALUES ('Series', 2026, 1, 'Publisher', 1),
			('Series', 2026, 2, 'Publisher', 0);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	list, err := listSeries(ctx, db, &ComicSeriesListInput{})
	if err != nil {
		t.Fatalf("listSeries: %v", err)
	}
	if len(list.Body) != 1 {
		t.Fatalf("series count = %d; want 1", len(list.Body))
	}
	if list.Body[0].Progress != 0.5 || list.Body[0].ReadCount != 1 || list.Body[0].EntryCount != 2 {
		t.Fatalf("series stats = %#v; want progress .5, read 1, entries 2", list.Body[0])
	}

	detail, err := updateSeriesFavorite(ctx, db, list.Body[0].ID, true)
	if err != nil {
		t.Fatalf("updateSeriesFavorite: %v", err)
	}
	if !detail.Body.Favorite {
		t.Fatal("series favorite was not saved")
	}
	if len(detail.Body.Comics) != 2 {
		t.Fatalf("detail comics = %d; want 2", len(detail.Body.Comics))
	}
}

func TestSeriesSyncDoesNotFailWhenPruneFails(t *testing.T) {
	ctx := context.Background()
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	if _, err := db.Exec(`
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
		INSERT INTO series (name, series_year)
		VALUES ('Stale', 2026);
		INSERT INTO comics (series, series_year, issue, publisher)
		VALUES ('Live', 2026, 1, 'Publisher');
		CREATE TRIGGER fail_series_prune
		BEFORE DELETE ON series
		BEGIN
			SELECT RAISE(FAIL, 'prune blocked');
		END;
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	list, err := listSeries(ctx, db, &ComicSeriesListInput{})
	if err != nil {
		t.Fatalf("listSeries: %v", err)
	}
	if len(list.Body) != 2 {
		t.Fatalf("series count = %d; want live plus stale rows", len(list.Body))
	}
}

func TestDocsConfigAndRouteMetadata(t *testing.T) {
	router := chi.NewRouter()
	api := humachi.New(router, DocsConfig())
	RegisterComicRoutes(api, nil, nil)
	RegisterSeriesRoutes(api, nil, metron.New(metron.Config{}), nil, newMetronImportJobStore())
	RegisterCharacterRoutes(api, nil)
	RegisterReadingOrderRoutes(api, nil)
	RegisterArcRoutes(api, nil)
	RegisterMetronRoutes(api, nil, metron.New(metron.Config{}), nil, newMetronImportJobStore())

	openAPI := api.OpenAPI()
	if openAPI.Info.Description == "" {
		t.Fatal("OpenAPI description is empty")
	}
	if len(openAPI.Tags) != 6 {
		t.Fatalf("len(tags) = %d; want 6", len(openAPI.Tags))
	}

	listComics := openAPI.Paths["/comics"].Get
	if len(listComics.Tags) != 1 || listComics.Tags[0] != tagComics {
		t.Fatalf("list comics tags = %#v; want Comics tag", listComics.Tags)
	}
	if _, ok := listComics.Responses["400"]; !ok {
		t.Fatal("list comics response docs missing 400 error")
	}

	listCharacters := openAPI.Paths["/characters"].Get
	if len(listCharacters.Tags) != 1 || listCharacters.Tags[0] != tagCharacters {
		t.Fatalf("list characters tags = %#v; want Characters tag", listCharacters.Tags)
	}

	listSeries := openAPI.Paths["/series"].Get
	if len(listSeries.Tags) != 1 || listSeries.Tags[0] != tagSeries {
		t.Fatalf("list series tags = %#v; want Series tag", listSeries.Tags)
	}

	listArcs := openAPI.Paths["/arcs"].Get
	if len(listArcs.Tags) != 1 || listArcs.Tags[0] != tagArcs {
		t.Fatalf("list arcs tags = %#v; want Arcs tag", listArcs.Tags)
	}

	searchMetronArcs := openAPI.Paths["/metron/arcs"].Get
	if len(searchMetronArcs.Tags) != 1 || searchMetronArcs.Tags[0] != tagMetron {
		t.Fatalf("search Metron arcs tags = %#v; want Metron tag", searchMetronArcs.Tags)
	}

	importSeries := openAPI.Paths["/metron/series/{id}/import"].Post
	if len(importSeries.Tags) != 1 || importSeries.Tags[0] != tagMetron {
		t.Fatalf("import series tags = %#v; want Metron tag", importSeries.Tags)
	}
	if _, ok := importSeries.Responses["429"]; !ok {
		t.Fatal("import series response docs missing 429 error")
	}
}

func TestMountedDocsLoadMountedOpenAPISpec(t *testing.T) {
	router := chi.NewRouter()
	apiRouter := chi.NewRouter()
	router.Mount("/api", apiRouter)
	humachi.New(apiRouter, DocsConfig())

	docs := httptest.NewRecorder()
	router.ServeHTTP(docs, httptest.NewRequest(http.MethodGet, "/api/docs", nil))
	if docs.Code != http.StatusOK {
		t.Fatalf("docs status = %d; want 200", docs.Code)
	}
	if !strings.Contains(docs.Body.String(), `apiDescriptionUrl="/api/openapi.yaml"`) {
		t.Fatalf("docs body did not reference mounted OpenAPI spec: %s", docs.Body.String())
	}

	spec := httptest.NewRecorder()
	router.ServeHTTP(spec, httptest.NewRequest(http.MethodGet, "/api/openapi.yaml", nil))
	if spec.Code != http.StatusOK {
		t.Fatalf("spec status = %d; want 200", spec.Code)
	}
}
