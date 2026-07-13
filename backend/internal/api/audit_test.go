package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestAuditMiddlewareRecordsOnlySuccessfulMutations(t *testing.T) {
	db, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE audit_events (id INTEGER PRIMARY KEY, user_id INTEGER, method TEXT, path TEXT, status_code INTEGER, occurred_at TEXT DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}

	handler := AuditMiddleware(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/failed" {
			http.Error(w, "failed", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, request := range []struct {
		method string
		path   string
	}{{http.MethodGet, "/comics"}, {http.MethodPatch, "/failed"}, {http.MethodPost, "/readingOrders"}} {
		req := httptest.NewRequest(request.method, request.path, nil)
		req = req.WithContext(context.WithValue(req.Context(), contextUserIDKey{}, 7))
		handler.ServeHTTP(httptest.NewRecorder(), req)
	}

	var events []AuditEvent
	if err := db.Select(&events, `SELECT id, user_id, method, path, status_code, occurred_at, '' AS user_name FROM audit_events`); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].UserID == nil || *events[0].UserID != 7 || events[0].Method != http.MethodPost || events[0].Path != "/readingOrders" {
		t.Fatalf("audit events = %#v; want one successful POST by user 7", events)
	}
}
