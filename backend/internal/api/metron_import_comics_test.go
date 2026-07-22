package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Lofter1/ComicHero/backend/internal/metron"
	"github.com/danielgtaylor/huma/v2"
)

func TestSearchMetronIssuesPrefersComicVineID(t *testing.T) {
	var requestURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"id":77,"cv_id":9001}]}`))
	}))
	defer server.Close()

	issues, err := searchMetronIssues(context.Background(), metron.New(metron.Config{BaseURL: server.URL}), &MetronIssueListInput{
		Query:       "ignored title",
		Series:      "ignored series",
		Issue:       "1",
		ComicVineID: 9001,
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestURL != "/issue/?cv_id=9001" {
		t.Fatalf("request URL = %q; want exact Comic Vine lookup", requestURL)
	}
	if len(issues) != 1 || issues[0].ID != 77 {
		t.Fatalf("issues = %+v; want Metron issue 77", issues)
	}
}

func TestUpdateComicFromMetronReportsExistingIssueLink(t *testing.T) {
	db := newMetronImportTestDB(t)
	ctx := testUserContext()
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, comic_vine_id)
		VALUES ('Target', 2026, '1', '', 9001);
		INSERT INTO comics (series, series_year, issue, publisher, metron_issue_id)
		VALUES ('Already linked', 2020, '7', '', 77);
	`); err != nil {
		t.Fatal(err)
	}

	_, err := updateComicFromMetron(ctx, db, nil, nil, 1, metron.Issue{ID: 77, ComicVineID: 9001})
	if err == nil {
		t.Fatal("expected Metron issue link conflict")
	}
	statusErr, ok := err.(interface{ GetStatus() int })
	if !ok || statusErr.GetStatus() != http.StatusConflict {
		t.Fatalf("error = %T %v; want HTTP 409 conflict", err, err)
	}
	want := "Metron issue 77 is already linked to Already linked (2020) #7 (comic 2). Merge the duplicate comics or choose another Metron issue."
	if err.Error() != want {
		t.Fatalf("error = %q; want %q", err, want)
	}
	problem, ok := err.(*huma.ErrorModel)
	if !ok || problem.Type != metronIssueAlreadyLinkedProblem {
		t.Fatalf("error problem type = %#v; want %q", problem, metronIssueAlreadyLinkedProblem)
	}
}

func TestMergeComicLinkedToMetronIssueKeepsSelectedComic(t *testing.T) {
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		UPDATE users SET is_admin = 1 WHERE id = 1;
		INSERT INTO comics (id, series, series_year, issue, publisher)
		VALUES (1, 'Selected', 2026, '1', '');
		INSERT INTO comics (id, series, series_year, issue, publisher, metron_issue_id)
		VALUES (2, 'Linked duplicate', 2020, '7', 'Publisher', 77);
	`); err != nil {
		t.Fatal(err)
	}

	merged, err := mergeComicLinkedToMetronIssue(deletionUserContext(1), db, 1, 77)
	if err != nil {
		t.Fatal(err)
	}
	if merged == nil || merged.Body.ID != 1 {
		t.Fatalf("merged comic = %#v; want selected comic 1", merged)
	}
	if merged.Body.MetronIssueID == nil || *merged.Body.MetronIssueID != 77 {
		t.Fatalf("Metron issue ID = %#v; want 77", merged.Body.MetronIssueID)
	}
	assertComicMergeCount(t, db, `SELECT COUNT(*) FROM comics WHERE id = 2`, 0)
}

func TestImportMetronComicReusesExistingMetronComic(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	issue := metron.Issue{
		ID:          101,
		ComicVineID: 9001,
		Series:      "Series",
		SeriesYear:  2026,
		Issue:       "1",
		Publisher:   "Publisher",
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
	if first.Body.ComicVineID == nil || *first.Body.ComicVineID != 9001 {
		t.Fatalf("Comic Vine ID = %#v; want 9001", first.Body.ComicVineID)
	}
}

func TestImportMetronComicReusesComicVineMatchAndAttachesMetronID(t *testing.T) {
	ctx := testUserContext()
	db := newMetronImportTestDB(t)
	if _, err := db.Exec(`
		INSERT INTO comics (series, series_year, issue, publisher, comic_vine_id)
		VALUES ('Older local title', 1999, '7', '', 81234)
	`); err != nil {
		t.Fatalf("seed Comic Vine comic: %v", err)
	}

	comic, err := importMetronComic(ctx, db, nil, nil, metron.Issue{
		ID:          404,
		ComicVineID: 81234,
		Series:      "Canonical title",
		SeriesYear:  2026,
		Issue:       "1",
		Publisher:   "Publisher",
	})
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if comic.Body.ID != 1 {
		t.Fatalf("comic ID = %d; want existing comic 1", comic.Body.ID)
	}
	if comic.Body.MetronIssueID == nil || *comic.Body.MetronIssueID != 404 {
		t.Fatalf("Metron issue ID = %#v; want 404", comic.Body.MetronIssueID)
	}
	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM comics`); err != nil {
		t.Fatalf("count comics: %v", err)
	}
	if count != 1 {
		t.Fatalf("comic count = %d; want 1", count)
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
		INSERT INTO user_characters (character_id, user_id, favorite)
		VALUES (1, (SELECT id FROM users WHERE name = 'Default'), 1);
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
		VALUES (1, 'Old Hero', 'Old description', 'old-image', 1, 301);
		INSERT INTO user_characters (character_id, user_id, favorite)
		VALUES (1, (SELECT id FROM users WHERE name = 'Default'), 1)
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
