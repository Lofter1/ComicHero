package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	comicdb "github.com/Lofter1/ComicHero/backend/internal/db"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestListCharactersProductionSizedDataset(t *testing.T) {
	database, err := comicdb.Open(filepath.Join(t.TempDir(), "characters.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer func() { _ = database.Close() }()

	if _, err := database.Exec(`
		WITH RECURSIVE sequence(value) AS (
			VALUES (1)
			UNION ALL
			SELECT value + 1 FROM sequence WHERE value < 500
		)
		INSERT INTO characters (id, name, metron_character_id)
		SELECT value, printf('Character %04d', value), 100000 + value
		FROM sequence;

		INSERT INTO character_aliases (character_id, alias)
		SELECT id, printf('Alias %04d', id)
		FROM characters;

		WITH RECURSIVE sequence(value) AS (
			VALUES (1)
			UNION ALL
			SELECT value + 1 FROM sequence WHERE value < 50000
		)
		INSERT INTO comics (id, series, issue, publisher)
		SELECT value, printf('Series %04d', (value - 1) / 100), CAST((value % 100) + 1 AS TEXT), 'Publisher'
		FROM sequence;

		INSERT INTO comic_characters (comic_id, character_id)
		SELECT id, ((id - 1) % 500) + 1
		FROM comics;

		INSERT INTO user_comics (comic_id, user_id, read)
		SELECT id, 1, id % 2
		FROM comics;

		INSERT INTO user_characters (character_id, user_id, started_at, favorite)
		SELECT id, 1, CASE WHEN id % 3 = 0 THEN CURRENT_TIMESTAMP END, id % 2
		FROM characters;
	`); err != nil {
		t.Fatalf("seed production-sized character data: %v", err)
	}

	router := chi.NewRouter()
	humaAPI := humachi.New(router, DocsConfig())
	RegisterCharacterRoutes(humaAPI, database)
	request := httptest.NewRequest(http.MethodGet, "/characters?limit=100", nil)
	request = request.WithContext(testUserContext())
	recorder := httptest.NewRecorder()

	started := time.Now()
	router.ServeHTTP(recorder, request)
	elapsed := time.Since(started)
	if elapsed > 2*time.Second {
		t.Fatalf("GET /characters took %s; want at most 2s", elapsed)
	}
	t.Logf("GET /characters completed against production-sized data in %s", elapsed)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /characters status = %d; want 200: %s", recorder.Code, recorder.Body.String())
	}

	var characters []Character
	if err := json.NewDecoder(recorder.Body).Decode(&characters); err != nil {
		t.Fatalf("decode characters response: %v", err)
	}
	if len(characters) != 100 {
		t.Fatalf("characters = %d; want one 100-row page", len(characters))
	}
	if total, hasMore := recorder.Header().Get("X-Total-Count"), recorder.Header().Get("X-Has-More"); total != "500" || hasMore != "true" {
		t.Fatalf("pagination = total %s, has more %s; want 500, true", total, hasMore)
	}
	first := characters[0]
	if first.AppearanceCount != 100 {
		t.Fatalf("first character appearances = %d; want 100", first.AppearanceCount)
	}
	if len(first.Aliases) != 1 || first.Aliases[0] != "Alias 0001" {
		t.Fatalf("first character aliases = %#v; want Alias 0001", first.Aliases)
	}
}
