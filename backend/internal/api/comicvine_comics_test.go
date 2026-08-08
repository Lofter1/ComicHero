package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Lofter1/ComicHero/backend/comicvine"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestUpdateComicFromComicVineOverwritesExistingFields(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, cover_image, description, comic_vine_id)
		VALUES ('Old Series', 1990, '0', 'Old Publisher', '1990-01-01', '', 'Old description', 501)
	`); err != nil {
		t.Fatal(err)
	}

	issue := comicvine.Issue{
		ID:          501,
		IssueNumber: "1",
		CoverDate:   "1999-06-01",
		Description: "First appearance.",
	}
	volume := &comicvine.Volume{
		ID:        900,
		Name:      "New Series",
		StartYear: "1999",
		Publisher: comicvine.Publisher{Name: "New Publisher"},
	}

	output, err := updateComicFromComicVine(ctx, db, &CoverCache{}, 1, issue, volume)
	if err != nil {
		t.Fatal(err)
	}
	if output.Body.Series != "New Series" || output.Body.SeriesYear != 1999 || output.Body.Issue != "1" {
		t.Fatalf("series metadata = %+v; want overwritten from Comic Vine", output.Body.Comic)
	}
	if output.Body.Publisher != "New Publisher" {
		t.Fatalf("publisher = %q; want New Publisher", output.Body.Publisher)
	}
	if output.Body.CoverDate != "1999-06-01" || output.Body.Description != "First appearance." {
		t.Fatalf("cover date/description not overwritten: %+v", output.Body.Comic)
	}

	var syncedAt string
	if err := db.Get(&syncedAt, `SELECT comic_vine_synced_at FROM comics WHERE id = 1`); err != nil {
		t.Fatal(err)
	}
	if syncedAt == "" {
		t.Fatal("expected comic_vine_synced_at to be stamped")
	}
}

func TestUpdateComicFromComicVineKeepsExistingValuesWhenVolumeLookupFailed(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, cover_image, description, comic_vine_id)
		VALUES ('Kept Series', 1990, '0', 'Kept Publisher', '1990-01-01', '', 'Kept description', 501)
	`); err != nil {
		t.Fatal(err)
	}

	issue := comicvine.Issue{ID: 501, IssueNumber: "1", CoverDate: "1999-06-01", Description: "Updated."}

	// No volume passed (as if the volume lookup failed) — series and publisher
	// should be left untouched, but the fields Comic Vine did return should
	// still be applied.
	output, err := updateComicFromComicVine(ctx, db, &CoverCache{}, 1, issue, nil)
	if err != nil {
		t.Fatal(err)
	}
	if output.Body.Series != "Kept Series" || output.Body.Publisher != "Kept Publisher" {
		t.Fatalf("series/publisher = %+v; want preserved when volume lookup failed", output.Body.Comic)
	}
	if output.Body.CoverDate != "1999-06-01" || output.Body.Description != "Updated." {
		t.Fatalf("cover date/description = %+v; want applied from the issue", output.Body.Comic)
	}
}

func TestRefreshComicFromComicVineHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/issue/4000-501/":
			_ = json.NewEncoder(w).Encode(comicvine.SingleIssueResponse{
				StatusCode: 1,
				Results: comicvine.Issue{
					ID:          501,
					IssueNumber: "1",
					CoverDate:   "1999-06-01",
					Description: "First appearance.",
					Volume:      &comicvine.Volume{ID: 900},
				},
			})
		case "/volume/4050-900/":
			_ = json.NewEncoder(w).Encode(comicvine.SingleVolumeResponse{
				StatusCode: 1,
				Results: comicvine.Volume{
					ID:        900,
					Name:      "New Series",
					StartYear: "1999",
					Publisher: comicvine.Publisher{Name: "New Publisher"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, cover_date, cover_image, description, comic_vine_id)
		VALUES
			('Old Series', 1990, '0', 'Old Publisher', '', '', '', 501),
			('Unlinked', 2020, '1', '', '', '', '', NULL)
	`); err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	humaAPI := humachi.New(router, DocsConfig())
	client := comicvine.NewClient("test-key", comicvine.WithBaseURL(server.URL))
	RegisterComicVineComicRoutes(humaAPI, db, client, &CoverCache{})

	// Linked comic: full refresh succeeds and overwrites fields.
	refresh := httptest.NewRequest(http.MethodPatch, "/comics/1/comicvine", nil).WithContext(testUserContext())
	refreshRecorder := httptest.NewRecorder()
	router.ServeHTTP(refreshRecorder, refresh)
	if refreshRecorder.Code != http.StatusOK {
		t.Fatalf("PATCH /comics/1/comicvine status = %d; want 200: %s", refreshRecorder.Code, refreshRecorder.Body.String())
	}
	var refreshed ComicDetail
	if err := json.NewDecoder(refreshRecorder.Body).Decode(&refreshed); err != nil {
		t.Fatal(err)
	}
	if refreshed.Series != "New Series" || refreshed.Publisher != "New Publisher" {
		t.Fatalf("refreshed comic = %+v; want fields from Comic Vine", refreshed)
	}

	// Unlinked comic: rejected before any Comic Vine call is made.
	unlinked := httptest.NewRequest(http.MethodPatch, "/comics/2/comicvine", nil).WithContext(testUserContext())
	unlinkedRecorder := httptest.NewRecorder()
	router.ServeHTTP(unlinkedRecorder, unlinked)
	if unlinkedRecorder.Code != http.StatusBadRequest {
		t.Fatalf("PATCH /comics/2/comicvine (unlinked) status = %d; want 400: %s", unlinkedRecorder.Code, unlinkedRecorder.Body.String())
	}
}

func TestRefreshComicFromComicVineRequiresAPIKey(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, comic_vine_id) VALUES ('Linked', 2020, '1', '', 501)
	`); err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	humaAPI := humachi.New(router, DocsConfig())
	client := comicvine.NewClient("") // no API key configured
	RegisterComicVineComicRoutes(humaAPI, db, client, &CoverCache{})

	request := httptest.NewRequest(http.MethodPatch, "/comics/1/comicvine", nil).WithContext(testUserContext())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d; want 400 when no API key is configured: %s", recorder.Code, recorder.Body.String())
	}
}
