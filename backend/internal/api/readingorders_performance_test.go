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

func TestListReadingOrdersProductionSizedDataset(t *testing.T) {
	database, err := comicdb.Open(filepath.Join(t.TempDir(), "reading-orders.db"))
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if _, err := database.Exec(`
		WITH RECURSIVE sequence(value) AS (
			VALUES (1)
			UNION ALL
			SELECT value + 1 FROM sequence WHERE value < 500
		)
		INSERT INTO reading_orders (id, name, description, author_user_id)
		SELECT value, printf('Reading order %04d', value), 'Description', 1
		FROM sequence;

		WITH RECURSIVE sequence(value) AS (
			VALUES (1)
			UNION ALL
			SELECT value + 1 FROM sequence WHERE value < 50000
		)
		INSERT INTO comics (id, series, series_year, issue, publisher, cover_image)
		SELECT value, 'Series', 2026, CAST(value AS TEXT), 'Publisher',
			CASE WHEN value % 10 = 0 THEN printf('/covers/%d.jpg', value) ELSE '' END
		FROM sequence;

		WITH RECURSIVE sequence(value) AS (
			VALUES (1)
			UNION ALL
			SELECT value + 1 FROM sequence WHERE value < 50000
		)
		INSERT INTO reading_order_comics (reading_order_id, comic_id, position)
		SELECT ((value - 1) / 100) + 1, value, ((value - 1) % 100) + 1
		FROM sequence;

		INSERT INTO user_comics (comic_id, user_id, read)
		SELECT id, 1, id % 2 FROM comics;
	`); err != nil {
		t.Fatalf("seed production-sized reading-order data: %v", err)
	}

	router := chi.NewRouter()
	humaAPI := humachi.New(router, DocsConfig())
	RegisterReadingOrderRoutes(humaAPI, database, nil)
	request := httptest.NewRequest(http.MethodGet, "/readingOrders?limit=50", nil)
	request = request.WithContext(testUserContext())
	recorder := httptest.NewRecorder()

	started := time.Now()
	router.ServeHTTP(recorder, request)
	elapsed := time.Since(started)
	if elapsed > 2*time.Second {
		t.Fatalf("GET /readingOrders took %s; want at most 2s", elapsed)
	}
	t.Logf("GET /readingOrders completed against production-sized data in %s", elapsed)
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /readingOrders status = %d; want 200: %s", recorder.Code, recorder.Body.String())
	}

	var readingOrders []ReadingOrder
	if err := json.NewDecoder(recorder.Body).Decode(&readingOrders); err != nil {
		t.Fatalf("decode reading orders response: %v", err)
	}
	if len(readingOrders) != 50 {
		t.Fatalf("reading orders = %d; want one 50-row page", len(readingOrders))
	}
	if total, hasMore := recorder.Header().Get("X-Total-Count"), recorder.Header().Get("X-Has-More"); total != "500" || hasMore != "true" {
		t.Fatalf("pagination = total %s, has more %s; want 500, true", total, hasMore)
	}
	if readingOrders[0].Progress != 0.5 {
		t.Fatalf("first reading order progress = %v; want 0.5", readingOrders[0].Progress)
	}
}
