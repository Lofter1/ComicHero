package api

import (
	"context"
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
		"ORDER BY c.series, c.series_year, c.issue",
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("query missing %q: %s", fragment, query)
		}
	}
	if len(args) != 9 {
		t.Fatalf("len(args) = %d; want 9", len(args))
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
			issue INTEGER NOT NULL,
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

func TestDocsConfigAndRouteMetadata(t *testing.T) {
	router := chi.NewRouter()
	api := humachi.New(router, DocsConfig())
	RegisterComicRoutes(api, nil, nil)
	RegisterSeriesRoutes(api, nil, metron.New(metron.Config{}), nil, newMetronImportJobStore())
	RegisterCharacterRoutes(api, nil)
	RegisterReadingOrderRoutes(api, nil)
	RegisterMetronRoutes(api, nil, metron.New(metron.Config{}), nil, newMetronImportJobStore())

	openAPI := api.OpenAPI()
	if openAPI.Info.Description == "" {
		t.Fatal("OpenAPI description is empty")
	}
	if len(openAPI.Tags) != 5 {
		t.Fatalf("len(tags) = %d; want 5", len(openAPI.Tags))
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

	importSeries := openAPI.Paths["/metron/series/{id}/import"].Post
	if len(importSeries.Tags) != 1 || importSeries.Tags[0] != tagMetron {
		t.Fatalf("import series tags = %#v; want Metron tag", importSeries.Tags)
	}
	if _, ok := importSeries.Responses["429"]; !ok {
		t.Fatal("import series response docs missing 429 error")
	}
}
